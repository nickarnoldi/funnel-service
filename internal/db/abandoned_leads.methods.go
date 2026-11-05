package db

import (
	"fmt"
	"log"

	"github.com/jackc/pgx/v5/pgtype"
)

func (lead *AbandonedLead) Save(store *Store) error {
	if lead.ID == 0 {
		newLead, err := store.Queries.CreateAbandonedLead(store.Ctx, CreateAbandonedLeadParams{
			OfferID:            lead.OfferID,
			OfferURLID:         lead.OfferURLID,
			ClickID:            lead.ClickID,
			StepNumber:         lead.StepNumber,
			IPAddress:          lead.IPAddress,
			TransactionID:      lead.TransactionID,
			AffiliateID:        lead.AffiliateID,
			CampaignID:         lead.CampaignID,
			PhoneNumber:        lead.PhoneNumber,
			Email:              lead.Email,
			JornayaID:          lead.JornayaID,
			TrustedformID:      lead.TrustedformID,
			IsBot:              lead.IsBot,
			IsMobile:           lead.IsMobile,
			CountryCode:        lead.CountryCode,
			RegionCode:         lead.RegionCode,
			City:               lead.City,
			PostalCode:         lead.PostalCode,
			DateOfBirth:        lead.DateOfBirth,
			EmploymentStatus:   lead.EmploymentStatus,
			GrossMonthlyIncome: lead.GrossMonthlyIncome,
			AnuraScore:         lead.AnuraScore,
			HoneypotFilled:     lead.HoneypotFilled,
			SessionLength:      lead.SessionLength,
			LandingPageURL:     lead.LandingPageURL,
			Referrer:           lead.Referrer,
			Cookies:            lead.Cookies,
			Body:               lead.Body,
		})
		if err != nil {
			return fmt.Errorf("failed to create abandoned lead: %w", err)
		}

		*lead = newLead
	} else {
		updatedLead, err := store.Queries.UpdateAbandonedLead(store.Ctx, UpdateAbandonedLeadParams{
			ID:                 lead.ID,
			OfferID:            lead.OfferID,
			OfferURLID:         lead.OfferURLID,
			ClickID:            lead.ClickID,
			StepNumber:         lead.StepNumber,
			IPAddress:          lead.IPAddress,
			TransactionID:      lead.TransactionID,
			AffiliateID:        lead.AffiliateID,
			CampaignID:         lead.CampaignID,
			PhoneNumber:        lead.PhoneNumber,
			Email:              lead.Email,
			JornayaID:          lead.JornayaID,
			TrustedformID:      lead.TrustedformID,
			IsBot:              lead.IsBot,
			IsMobile:           lead.IsMobile,
			CountryCode:        lead.CountryCode,
			RegionCode:         lead.RegionCode,
			City:               lead.City,
			PostalCode:         lead.PostalCode,
			DateOfBirth:        lead.DateOfBirth,
			EmploymentStatus:   lead.EmploymentStatus,
			GrossMonthlyIncome: lead.GrossMonthlyIncome,
			AnuraScore:         lead.AnuraScore,
			HoneypotFilled:     lead.HoneypotFilled,
			SessionLength:      lead.SessionLength,
			LandingPageURL:     lead.LandingPageURL,
			Referrer:           lead.Referrer,
			Cookies:            lead.Cookies,
			Body:               lead.Body,
		})
		if err != nil {
			log.Printf("Error updating abandoned lead %d: %v", lead.ID, err)
		}

		*lead = updatedLead
	}

	if lead.ID == 0 {
		return fmt.Errorf("failed to save abandoned lead, ID is still 0")
	}

	return nil
}

const TransactionIDMinLength = 30

type AbandonedLeadTokenParams struct {
	OfferID       string
	OfferURLID    string
	TransactionID string
	AffiliateID   string
	CampaignID    string
}

func (params *AbandonedLeadTokenParams) UpdateTransactionID(value string) bool {
	if len(value) >= TransactionIDMinLength {
		params.TransactionID = value
		return true
	}
	return false
}

func (params *AbandonedLeadTokenParams) MergeOtherParams(other AbandonedLeadTokenParams) {
	if other.OfferID != "" && params.OfferID == "" {
		params.OfferID = other.OfferID
	}

	if other.OfferURLID != "" && params.OfferURLID == "" {
		params.OfferURLID = other.OfferURLID
	}

	if other.TransactionID != "" && params.TransactionID == "" {
		params.TransactionID = other.TransactionID
	}

	if other.AffiliateID != "" && params.AffiliateID == "" {
		params.AffiliateID = other.AffiliateID
	}

	if other.CampaignID != "" && params.CampaignID == "" {
		params.CampaignID = other.CampaignID
	}
}

func (params *AbandonedLeadTokenParams) CompareOffer(offer Offer, externalID pgtype.Int8) bool {
	matchesOfferID := !externalID.Valid && params.OfferID != "" && params.OfferID == fmt.Sprintf("%d", offer.ID)
	matchesExternalID := externalID.Valid && params.OfferID != "" && params.OfferID == fmt.Sprintf("%d", externalID.Int64)
	if params.OfferID != "" && (matchesExternalID || matchesOfferID) {
		return true
	}

	return false
}

func (params *AbandonedLeadTokenParams) CompareOfferURL(offerURL OfferURL, externalID pgtype.Int8) bool {
	matchesOfferURLID := !externalID.Valid && params.OfferURLID != "" && params.OfferURLID == fmt.Sprintf("%d", offerURL.ID)
	matchesOfferURLExtID := externalID.Valid && params.OfferURLID != "" && params.OfferURLID == fmt.Sprintf("%d", externalID.Int64)
	if params.OfferURLID != "" && (matchesOfferURLID || matchesOfferURLExtID) {
		return true
	}

	return false
}
