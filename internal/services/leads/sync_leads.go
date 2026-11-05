package leads

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/nickarnoldi/funnel-service/internal/db"
	"github.com/nickarnoldi/funnel-service/internal/enum"
	"github.com/nickarnoldi/funnel-service/internal/server/api/handler/requests"
	"github.com/nickarnoldi/funnel-service/internal/services/everflow"
)

func SyncLeadsByDateRange(ctx context.Context, store *db.Store, startDate, endDate time.Time) error {
	parsedStartDate, parsedEndDate, err := store.GenerateSafeDateRange(startDate, endDate)
	if err != nil {
		return fmt.Errorf("failed to generate safe date range: %w", err)
	}

	fmt.Printf("Normalizing Leads from %s to %s\n", parsedStartDate, parsedEndDate)

	var startRange, endRange pgtype.Timestamptz
	startRange.Scan(parsedStartDate)
	endRange.Scan(parsedEndDate)

	var limit int = 100
	var total int

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	// paginate the results
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			fmt.Printf("Processing leads from %s to %s; total processed %d...\n", parsedStartDate, parsedEndDate, total)
		default:
		}

		// todo: Switch away from offset based pagination to keyset pagination for better performance on large datasets
		if leads, err := store.Queries.GetAbandonedLeadsByDateRange(ctx, db.GetAbandonedLeadsByDateRangeParams{
			From:   startRange,
			To:     endRange,
			Offset: db.Int(&total),
			Limit:  db.Int(&limit),
		}); err == nil {
			leadCount := len(leads)
			fmt.Printf("Running batch of %d of %d leads\n", leadCount, limit)

			for _, lead := range leads {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-ticker.C:
					fmt.Printf("Processing leads from %s to %s; total processed %d...\n", parsedStartDate, parsedEndDate, total)
				default:
				}

				fmt.Printf("Running lead %d\n", lead.ID)
				if result, err := syncAbandonedLead(ctx, store, lead); err != nil {
					if len(result.AuditLog) > 0 {
						log.Printf("Audit log for abandoned lead ID %d:\n", lead.ID)
						for _, logEntry := range result.AuditLog {
							log.Println(logEntry)
						}
					}
					log.Printf("Failed to sync abandoned lead ID %d: %v\n", lead.ID, err)
				}

				total++
			}

			if leadCount != limit {
				break
			}
		} else if err == pgx.ErrNoRows {
			break
		} else {
			return fmt.Errorf("failed to get normalized leads: %w", err)
		}

		time.Sleep(50 * time.Millisecond)
	}

	log.Printf("Total leads processed: %d\n", total)

	return nil
}

func SyncLeadsByIDs(ctx context.Context, store *db.Store, leadIDs []int64) error {
	for _, leadID := range leadIDs {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		lead, err := store.Queries.GetAbandonedLeadByID(ctx, leadID)
		if err != nil {
			return fmt.Errorf("failed to get abandoned lead by ID %d: %w", leadID, err)
		}

		if _, err := syncAbandonedLead(ctx, store, lead); err != nil {
			return fmt.Errorf("failed to sync abandoned lead ID %d: %w", leadID, err)
		}
	}

	return nil
}

func SyncLeadsByFunnelIDs(ctx context.Context, store *db.Store, funnelIDs []int64) error {
	var afterID pgtype.Int8

	fmt.Printf("Syncing abandoned leads for funnels ID %v\n", funnelIDs)

	count, err := store.Queries.CountAbandonedLeadsByFunnelID(ctx, funnelIDs)
	if err != nil {
		return fmt.Errorf("failed to count abandoned leads for funnels ID %v: %w", funnelIDs, err)
	}

	fmt.Printf("Total abandoned leads to sync: %d\n", count)

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("last processed ID %d", afterID.Int64)
		case <-ticker.C:
			fmt.Printf("Syncing abandoned leads for funnels ID %v; last worked ID %d...\n", funnelIDs, afterID.Int64)
		default:
		}

		leads, err := store.Queries.GetAbandonedLeadsByFunnelID(ctx, db.GetAbandonedLeadsByFunnelIDParams{
			FunnelIDs: funnelIDs,
			AfterID:   afterID,
		})
		if err == pgx.ErrNoRows || len(leads) == 0 {
			break
		} else if err != nil {
			return fmt.Errorf("failed to get abandoned leads for funnels ID %v: %w", funnelIDs, err)
		}

		for _, lead := range leads {
			select {
			case <-ctx.Done():
				return fmt.Errorf("last processed ID %d", afterID.Int64)
			case <-ticker.C:
				fmt.Printf("Syncing abandoned leads for funnels ID %v; last worked ID %d...\n", funnelIDs, afterID.Int64)
			default:
			}

			if _, err := syncAbandonedLead(ctx, store, lead); err != nil {
				return fmt.Errorf("failed to sync abandoned lead ID %d: %w", lead.ID, err)
			}
			afterID = db.BigInt(&lead.ID)
		}
	}

	return nil
}

type AbandonedLeadSyncResult struct {
	LeadID      int64
	ClickID     pgtype.Int8
	OfferID     int64
	OfferURLID  int64
	Transaction string
	AuditLog    []string
}

func syncAbandonedLead(ctx context.Context, store *db.Store, lead db.AbandonedLead) (AbandonedLeadSyncResult, error) {
	var leadResult AbandonedLeadSyncResult

	leadResult.AuditLog = append(leadResult.AuditLog, fmt.Sprintf("Processing Lead ID %d - TransactionID %s", lead.ID, lead.TransactionID))

	// Sanitize Body and Cookies to ensure valid UTF-8 before unmarshaling
	lead.Body = []byte(strings.ToValidUTF8(string(lead.Body), ""))
	lead.Cookies = []byte(strings.ToValidUTF8(string(lead.Cookies), ""))

	var body map[string]any
	if err := json.Unmarshal(lead.Body, &body); err != nil {
		return leadResult, fmt.Errorf("error unmarshalling lead body: %v", err)
	}

	bodyRaw, err := json.Marshal(body)
	if err != nil {
		return leadResult, fmt.Errorf("error marshalling lead body: %v", err)
	}

	var cookies map[string]string
	if err := json.Unmarshal(lead.Cookies, &cookies); err != nil {
		return leadResult, fmt.Errorf("error unmarshalling lead cookies: %v", err)
	}

	// Re-encode cookies to ensure the byte string size/format is consistent
	// {node:value} vs {node: value}
	leadCookies, err := json.Marshal(cookies)
	if err != nil {
		return leadResult, fmt.Errorf("error marshalling lead cookies: %v", err)
	}

	if lead.LandingPageURL == "" {
		log.Printf("Lead %d has an invalid landing page URL", lead.ID)
	}

	request := requests.AbandonedLeadRequest{
		LandingPageURL: lead.LandingPageURL,
		SessionLength:  lead.SessionLength,
		StepNumber:     lead.StepNumber,
		IPAddress:      lead.IPAddress,
		Referrer:       lead.Referrer,
		Cookies:        cookies,
		Body:           body,
	}

	result, err := request.Process(store)
	if err != nil || result.AbandonedLead == nil {
		return leadResult, fmt.Errorf("error processing lead %d: %v", lead.ID, err)
	}

	processedLead := result.AbandonedLead

	if !lead.ClickID.Valid && processedLead.TransactionID != "" && processedLead.ClickID.Valid {
		leadResult.AuditLog = append(leadResult.AuditLog, fmt.Sprintf("** Set Click ID: %d", processedLead.ClickID.Int64))
	} else if processedLead.ClickID.Int64 != lead.ClickID.Int64 {
		leadResult.AuditLog = append(leadResult.AuditLog, fmt.Sprintf("** Mismatched Click ID: %d was %d", processedLead.ClickID.Int64, lead.ClickID.Int64))
	}

	if processedLead.OfferID != lead.OfferID {
		leadResult.AuditLog = append(leadResult.AuditLog, fmt.Sprintf("** Mismatched Offer ID: %d was %d", processedLead.OfferID, lead.OfferID))
	}

	if processedLead.OfferURLID != lead.OfferURLID {
		leadResult.AuditLog = append(leadResult.AuditLog, fmt.Sprintf("** Mismatched Offer URL ID: %d was %d", processedLead.OfferURLID, lead.OfferURLID))
	}

	if processedLead.StepNumber != lead.StepNumber {
		leadResult.AuditLog = append(leadResult.AuditLog, fmt.Sprintf("** Mismatched Step Number: %d was %d", processedLead.StepNumber, lead.StepNumber))
	}

	if processedLead.TransactionID != lead.TransactionID {
		leadResult.AuditLog = append(leadResult.AuditLog, fmt.Sprintf("** Mismatched Transaction ID: %s was %s", processedLead.TransactionID, lead.TransactionID))
	}

	if processedLead.AffiliateID != lead.AffiliateID {
		leadResult.AuditLog = append(leadResult.AuditLog, fmt.Sprintf("** Mismatched Affiliate ID: '%s' was '%s'", processedLead.AffiliateID.String, lead.AffiliateID.String))
	}

	if processedLead.CampaignID != lead.CampaignID {
		leadResult.AuditLog = append(leadResult.AuditLog, fmt.Sprintf("** Mismatched Campaign ID: '%s' was '%s'", processedLead.CampaignID.String, lead.CampaignID.String))
	}

	if processedLead.AnuraScore != lead.AnuraScore {
		leadResult.AuditLog = append(leadResult.AuditLog, fmt.Sprintf("** Mismatched Anura score: '%s' was '%s'", processedLead.AnuraScore.String, lead.AnuraScore.String))
	}

	if processedLead.HoneypotFilled != lead.HoneypotFilled {
		leadResult.AuditLog = append(leadResult.AuditLog, fmt.Sprintf("** Mismatched Honeypot filled status: %t was %t", processedLead.HoneypotFilled, lead.HoneypotFilled))
	}

	if processedLead.SessionLength != lead.SessionLength {
		leadResult.AuditLog = append(leadResult.AuditLog, fmt.Sprintf("** Mismatched Session Length: %d was %d", processedLead.SessionLength, lead.SessionLength))
	}

	if processedLead.JornayaID != lead.JornayaID {
		leadResult.AuditLog = append(leadResult.AuditLog, fmt.Sprintf("** Mismatched Jornaya ID: '%s' was '%s'", processedLead.JornayaID.String, lead.JornayaID.String))
	}

	if processedLead.TrustedformID != lead.TrustedformID {
		leadResult.AuditLog = append(leadResult.AuditLog, fmt.Sprintf("** Mismatched TrustedForm ID: '%s' was '%s'", processedLead.TrustedformID.String, lead.TrustedformID.String))
	}

	if processedLead.IsBot != lead.IsBot {
		leadResult.AuditLog = append(leadResult.AuditLog, fmt.Sprintf("** Mismatched Bot status: %t was %t", processedLead.IsBot, lead.IsBot))
	}

	if processedLead.IsMobile != lead.IsMobile {
		leadResult.AuditLog = append(leadResult.AuditLog, fmt.Sprintf("** Mismatched Mobile status: %t was %t", processedLead.IsMobile, lead.IsMobile))
	}

	if processedLead.Email != lead.Email {
		leadResult.AuditLog = append(leadResult.AuditLog, fmt.Sprintf("** Mismatched Email: '%s' was '%s'", processedLead.Email.String, lead.Email.String))
	}

	if processedLead.PhoneNumber != lead.PhoneNumber {
		leadResult.AuditLog = append(leadResult.AuditLog, fmt.Sprintf("** Mismatched Phone Number: '%s' was '%s'", processedLead.PhoneNumber.String, lead.PhoneNumber.String))
	}

	if processedLead.IPAddress != lead.IPAddress {
		leadResult.AuditLog = append(leadResult.AuditLog, fmt.Sprintf("** Mismatched IP Address: '%s' was '%s'", processedLead.IPAddress, lead.IPAddress))
	}

	if processedLead.CountryCode != lead.CountryCode {
		leadResult.AuditLog = append(leadResult.AuditLog, fmt.Sprintf("** Mismatched Country Code: '%s' was '%s'", processedLead.CountryCode.String, lead.CountryCode.String))
	}

	if processedLead.RegionCode != lead.RegionCode {
		leadResult.AuditLog = append(leadResult.AuditLog, fmt.Sprintf("** Mismatched Region Code: '%s' was '%s'", processedLead.RegionCode.String, lead.RegionCode.String))
	}

	if processedLead.City != lead.City {
		leadResult.AuditLog = append(leadResult.AuditLog, fmt.Sprintf("** Mismatched City: '%s' was '%s'", processedLead.City.String, lead.City.String))
	}

	if processedLead.PostalCode != lead.PostalCode {
		leadResult.AuditLog = append(leadResult.AuditLog, fmt.Sprintf("** Mismatched Postal Code: '%s' was '%s'", processedLead.PostalCode.String, lead.PostalCode.String))
	}

	if processedLead.DateOfBirth.Time != lead.DateOfBirth.Time {
		leadResult.AuditLog = append(leadResult.AuditLog, fmt.Sprintf("** Mismatched Date of Birth: %s was %s", processedLead.DateOfBirth.Time, lead.DateOfBirth.Time))
	}

	if processedLead.EmploymentStatus != lead.EmploymentStatus {
		leadResult.AuditLog = append(leadResult.AuditLog, fmt.Sprintf("** Mismatched Employment Status: %d was %d", processedLead.EmploymentStatus.Int16, lead.EmploymentStatus.Int16))
	}

	newLeadIncome, _ := processedLead.GrossMonthlyIncome.Int64Value()
	leadIncome, _ := lead.GrossMonthlyIncome.Int64Value()
	if newLeadIncome != leadIncome {
		leadResult.AuditLog = append(leadResult.AuditLog, fmt.Sprintf("** Mismatched Gross Monthly Income: %v was %v", newLeadIncome, leadIncome))
	}

	if processedLead.LandingPageURL != lead.LandingPageURL {
		leadResult.AuditLog = append(leadResult.AuditLog, fmt.Sprintf("** Mismatched Landing Page URL: '%s' was '%s'", processedLead.LandingPageURL, lead.LandingPageURL))
	}

	if processedLead.Referrer != lead.Referrer {
		leadResult.AuditLog = append(leadResult.AuditLog, fmt.Sprintf("** Mismatched Referrer: '%s' was '%s'", processedLead.Referrer, lead.Referrer))
	}

	if len(processedLead.Cookies) != len(leadCookies) {
		leadResult.AuditLog = append(leadResult.AuditLog, fmt.Sprintf("** Mismatched Cookies length: %d was %d", len(processedLead.Cookies), len(leadCookies)))
	}

	if len(processedLead.Body) != len(bodyRaw) {
		leadResult.AuditLog = append(leadResult.AuditLog, fmt.Sprintf("** Mismatched Body length: %d was %d", len(processedLead.Body), len(lead.Body)))
	}

	// fmt.Printf("\n--- Audit log for abandoned lead ID %d ---\n", lead.ID)
	// for _, logEntry := range leadResult.AuditLog {
	// 	fmt.Println(logEntry)
	// }
	// fmt.Printf("--- End of audit log for abandoned lead ID %d ---\n\n", lead.ID)

	processedLead.ID = lead.ID
	if err := processedLead.Save(store); err != nil {
		return leadResult, fmt.Errorf("error updating abandoned lead %d: %v", lead.ID, err)
	}

	// Check the result.Step to see if we need to relink to the step
	leadStep, err := store.Queries.GetAbandonedLeadFunnelStepByAbandonedLeadID(ctx, lead.ID)
	switch err {
	case pgx.ErrNoRows:
		// If no lead step exists, we can create one
		if result.Step != nil {
			if _, err := store.Queries.CreateAbandonedLeadFunnelStep(ctx, db.CreateAbandonedLeadFunnelStepParams{
				AbandonedLeadID: lead.ID,
				FunnelStepID:    result.Step.ID,
			}); err != nil {
				return leadResult, fmt.Errorf("error creating abandoned lead funnel step for lead %d: %v", lead.ID, err)
			}
			leadResult.AuditLog = append(leadResult.AuditLog, fmt.Sprintf("** Created abandoned lead funnel step for lead %d to step ID %d", lead.ID, result.Step.ID))
		}
	case nil:
		// If we found a lead step, we can relink it
		if result.Step != nil {
			// check if is different than what we already have.
			if result.Step.ID != leadStep.FunnelStepID {
				if _, err := store.Queries.UpdateAbandonedLeadFunnelStep(ctx, db.UpdateAbandonedLeadFunnelStepParams{
					ID:              leadStep.ID,
					AbandonedLeadID: lead.ID,
					FunnelStepID:    result.Step.ID,
				}); err != nil {
					return leadResult, fmt.Errorf("error updating abandoned lead funnel step for lead %d: %v", lead.ID, err)
				}
				leadResult.AuditLog = append(leadResult.AuditLog, fmt.Sprintf("** Updated abandoned lead funnel step for lead %d to step ID %d", lead.ID, result.Step.ID))
			}
		} else {
			// If no step is found in the result, we can delete the existing step
			if err := store.Queries.DeleteAbandonedLeadFunnelStep(ctx, leadStep.ID); err != nil {
				return leadResult, fmt.Errorf("error deleting abandoned lead funnel step for lead %d: %v", lead.ID, err)
			}
			leadResult.AuditLog = append(leadResult.AuditLog, fmt.Sprintf("** Deleted abandoned lead funnel step for lead %d with step ID %d", lead.ID, leadStep.FunnelStepID))
		}
	default:
		return leadResult, fmt.Errorf("error getting abandoned lead funnel step for lead %d: %v", lead.ID, err)
	}

	return leadResult, nil
}

var transactionIDMap = map[string]bool{}

/**
 * Test: testSyncAbandonedLead processes an abandoned lead and compares the results
 */
func testSyncAbandonedLead(ctx context.Context, store *db.Store, lead db.AbandonedLead) (AbandonedLeadSyncResult, error) {
	var leadResult AbandonedLeadSyncResult

	leadResult.AuditLog = append(leadResult.AuditLog, fmt.Sprintf("Processing Lead ID %d - Click #%s", lead.ID, lead.TransactionID))

	// Sanitize Body and Cookies to ensure valid UTF-8 before unmarshaling
	lead.Body = []byte(strings.ToValidUTF8(string(lead.Body), ""))
	lead.Cookies = []byte(strings.ToValidUTF8(string(lead.Cookies), ""))

	var body map[string]any
	if err := json.Unmarshal(lead.Body, &body); err != nil {
		return leadResult, fmt.Errorf("error unmarshalling lead body: %v", err)
	}

	var cookies map[string]string
	if err := json.Unmarshal(lead.Cookies, &cookies); err != nil {
		return leadResult, fmt.Errorf("error unmarshalling lead cookies: %v", err)
	}

	if lead.LandingPageURL == "" {
		log.Printf("Lead %d has an invalid landing page URL", lead.ID)
	}

	request := requests.AbandonedLeadRequest{
		LandingPageURL: lead.LandingPageURL,
		SessionLength:  lead.SessionLength,
		StepNumber:     lead.StepNumber,
		IPAddress:      lead.IPAddress,
		Referrer:       lead.Referrer,
		Cookies:        cookies,
		Body:           body,
	}

	if !request.IsValid() {
		return AbandonedLeadSyncResult{}, fmt.Errorf("bad request")
	}

	result, err := request.TestBestMatches(store, false)
	if err != nil {
		return leadResult, fmt.Errorf("error finding best matches for lead %d: %v", lead.ID, err)
	}

	// check to see if the stored lead is different from the processed lead
	if lead.ClickID.Valid && result.TestMatch.Click != nil && lead.ClickID.Int64 == result.TestMatch.Click.ID {
		if lead.OfferID == result.TestMatch.Offer.ID {
			if lead.OfferURLID == result.TestMatch.OfferURL.ID {
				// fmt.Printf("Lead identical to TestMatch result for Lead ID %d\n", lead.ID)
				return leadResult, nil
			}
		}
	}

	if result.Score <= 2 || result.TestMatch.Offer == nil || result.TestMatch.OfferURL == nil || (result.TestMatch.Click == nil && lead.TransactionID != "") {
		fmt.Println("---")
		fmt.Printf("Lead ID %d - ", lead.ID)

		if result.Score <= 2 {
			fmt.Printf("Low Score: %d\n", result.Score)
		}
		if result.TestMatch.Offer == nil {
			fmt.Printf("TestMatch Offer is nil\n")
		}
		if result.TestMatch.OfferURL == nil {
			fmt.Printf("TestMatch OfferURL is nil\n")
		}
		if result.TestMatch.Click == nil && lead.TransactionID != "" {
			if transactionIDMap[lead.TransactionID] {
				fmt.Printf("TestMatch Click is nil, TransactionID: %s (already processed | %d total in cache)\n", lead.TransactionID, len(transactionIDMap))
				return leadResult, nil
			}

			transactionIDMap[lead.TransactionID] = true
			click, err := everflow.ImportRawClickByID(ctx, store, enum.NetworkEnum(result.TestMatch.Offer.NetworkID), lead.TransactionID)
			if err != nil {
				fmt.Printf("TestMatch Click is nil, TransactionID: %s\n", lead.TransactionID)
				return leadResult, nil
			}

			fmt.Printf("Imported Click for TransactionID %s: Click ID %d\n", lead.TransactionID, click.ID)
		}

		for _, logEntry := range result.Logs {
			fmt.Print(logEntry)
		}
		fmt.Println("--- Audit ---")
		for _, logEntry := range result.OfferMatch.Logs {
			fmt.Println(logEntry)
		}
		fmt.Println("--- Test Audit ---")
		for _, logEntry := range result.TestMatch.Logs {
			fmt.Println(logEntry)
		}
		fmt.Printf("---\n\n")
		os.Exit(0)
	}

	return leadResult, nil
}
