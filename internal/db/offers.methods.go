package db

import (
	"fmt"
	"log"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/nickarnoldi/funnel-service/internal/enum"
)

type FindBestOfferMatchParams struct {
	LandingPageURL string
	TransactionID  string
	OfferID        string
	OfferURLID     string
}

var testPageURL = regexp.MustCompile(`^(?:https?://)?([^/]+)/?([^/\?]*)(?:\??.*)?$`)

func (s *Store) FindBestOfferMatch(params FindBestOfferMatchParams) (OfferMatchResult, error) {
	if params.LandingPageURL == "" {
		return OfferMatchResult{}, fmt.Errorf("landing page URL is required")
	}

	landingPage, err := url.Parse(params.LandingPageURL)
	if err != nil {
		return OfferMatchResult{}, fmt.Errorf("invalid landing page URL: %s", params.LandingPageURL)
	}

	matchParams := FindBestOffersParams{
		PageURL: landingPage.String(),
	}

	if params.TransactionID != "" {
		matchParams.TransactionID.Scan(params.TransactionID)
	}
	if params.OfferID != "" {
		matchParams.ExternalOfferID.Scan(params.OfferID)
	}
	if params.OfferURLID != "" {
		matchParams.ExternalOfferURLID.Scan(params.OfferURLID)
	}

	matches, err := s.Queries.FindBestOffers(s.Ctx, matchParams)
	if err != nil {
		return OfferMatchResult{}, fmt.Errorf("error fetching offers by page URL: %w", err)
	}

	offerCount := len(matches)

	logs := make([]string, 0)

	if offerCount == 0 {
		logs = append(logs, fmt.Sprintf("Something like.. %s", testPageURL.ReplaceAllString(params.LandingPageURL, "%$1/$2%")))
		logs = append(logs, fmt.Sprintf("Params present: ClickID: %s, OfferID: %s, OfferURLID: %s", params.TransactionID, params.OfferID, params.OfferURLID))

		return OfferMatchResult{
			Logs: logs,
		}, fmt.Errorf("No offers found")
	}

	var bestMatchIndex int
	var bestMatches map[int]OfferMatchResult = make(map[int]OfferMatchResult)

	landingPageQuery := landingPage.Query()

	for index, offerMatch := range matches {
		matchLogs := make([]string, 0)
		matchResult := OfferMatchResult{
			Score:      1,
			Offer:      &offerMatch.Offer,
			OfferURL:   &offerMatch.OfferURL,
			Parameters: &AbandonedLeadTokenParams{},
		}

		offerMatchURL, err := url.Parse(offerMatch.OfferURL.DestinationURL)
		if err != nil {
			return bestMatches[bestMatchIndex], fmt.Errorf("invalid offer URL: %s", offerMatch.OfferURL.DestinationURL)
		}

		if offerMatchURL.Hostname() == landingPage.Hostname() {
			matchLogs = append(matchLogs, fmt.Sprintf(" +1 - Matched hostname: %s", offerMatchURL.Hostname()))
			matchResult.Score++
		}

		if offerMatchURL.EscapedPath() == landingPage.EscapedPath() {
			matchLogs = append(matchLogs, fmt.Sprintf(" +1 - Matched path: %s", offerMatchURL.Path))
			matchResult.Score++
		} else if len(offerMatchURL.EscapedPath()) > len(landingPage.EscapedPath()) {
			matchLogs = append(matchLogs, fmt.Sprintf(" -1 - Offer path is longer than Landing Page path: %s vs %s", offerMatchURL.EscapedPath(), landingPage.EscapedPath()))
			matchResult.Score--
		}

		if offerMatch.Offer.Status == int32(enum.OfferStatusActive) {
			matchLogs = append(matchLogs, " +1 - Offer is active")
			matchResult.Score++

			if offerMatch.OfferURL.Status == int32(enum.OfferURLStatusActive) {
				matchLogs = append(matchLogs, " +1 - Offer URL is active")
				matchResult.Score++
			}
		}

		offerMatchQuery := offerMatchURL.Query()

		// Compare the query parameters of the offer with the landing page query parameters
		for matchQueryKey, matchQueryValues := range offerMatchQuery {
			if landingPageQuery.Has(matchQueryKey) { // s1, ck, etc...
				landingPageQueryValues := landingPageQuery[matchQueryKey]
				for _, matchOfferQueryValue := range matchQueryValues {
					matchOfferQueryValue = strings.TrimSpace(matchOfferQueryValue)

					// check if using placeholders like {offer_id}, {{offer_id}} or __OFFER_ID__
					if placeholderTokenTest.MatchString(matchOfferQueryValue) {
						for _, currentLandingPageQueryValue := range landingPageQueryValues {
							currentLandingPageQueryValue = strings.TrimSpace(currentLandingPageQueryValue)
							if currentLandingPageQueryValue != "" {
								isNetworkMatch := offerMatch.Offer.CheckNetworkMatch(&currentLandingPageQueryValue)

								if strings.Contains(matchOfferQueryValue, "transaction_id") || strings.Contains(matchOfferQueryValue, "transaction.id") || strings.Contains(matchOfferQueryValue, "CLICKID") {
									matchResult.Parameters.UpdateTransactionID(currentLandingPageQueryValue)
								} else if strings.Contains(matchOfferQueryValue, "affiliate_id") || strings.Contains(matchOfferQueryValue, "affiliate.id") {
									matchResult.Parameters.AffiliateID = currentLandingPageQueryValue
								} else if strings.Contains(matchOfferQueryValue, "campaign_id") || strings.Contains(matchOfferQueryValue, "campaign.id") {
									matchResult.Parameters.CampaignID = currentLandingPageQueryValue
								} else if strings.Contains(matchOfferQueryValue, "offer_id") {
									matchResult.Parameters.OfferID = currentLandingPageQueryValue
								}

								if isNetworkMatch {
									matchLogs = append(matchLogs, fmt.Sprintf(" +2 - Matched placeholder query parameter + network %s: %s = %s", matchOfferQueryValue, matchQueryKey, currentLandingPageQueryValue))
									matchResult.Score++
								} else {
									matchLogs = append(matchLogs, fmt.Sprintf(" +1 - Matched placeholder query parameter %s: %s = %s", matchOfferQueryValue, matchQueryKey, currentLandingPageQueryValue))
								}
								matchResult.Score++
								break
							}
						}
					} else {
						if slices.Contains(landingPageQueryValues, matchOfferQueryValue) {
							isNetworkMatch := offerMatch.Offer.CheckNetworkMatch(&matchOfferQueryValue)

							switch matchQueryKey {
							case "transaction_id", "_ef_transaction_id":
								matchResult.Parameters.UpdateTransactionID(matchOfferQueryValue)
							case "ef_oid", "oid", "offer_id":
								matchResult.Parameters.OfferID = matchOfferQueryValue
							case "uid", "ef_uid", "offer_url_id":
								matchResult.Parameters.OfferURLID = matchOfferQueryValue
							case "affid", "aid", "affiliate_id":
								matchResult.Parameters.AffiliateID = matchOfferQueryValue
							}

							if isNetworkMatch {
								matchLogs = append(matchLogs, fmt.Sprintf(" +2 - Matched query parameter + network %s: %s = %s", matchQueryKey, matchOfferQueryValue, matchOfferQueryValue))
								matchResult.Score++
							} else {
								matchLogs = append(matchLogs, fmt.Sprintf(" +1 - Matched query parameter: %s = %s", matchQueryKey, matchOfferQueryValue))
							}
							matchResult.Score++
						}
					}
				}
			}
		}

		if matchResult.Parameters.CompareOffer(offerMatch.Offer, offerMatch.ExternalOfferID) {
			matchLogs = append(matchLogs, fmt.Sprintf(" +1 - Matched Offer ID: %s", matchResult.Parameters.OfferID))
			matchResult.Score++
		}

		if matchResult.Parameters.CompareOfferURL(offerMatch.OfferURL, offerMatch.ExternalOfferURLID) {
			matchLogs = append(matchLogs, fmt.Sprintf(" +1 - Matched Offer URL ID: %s", matchResult.Parameters.OfferURLID))
			matchResult.Score++
		}

		if matchResult.Parameters.TransactionID != "" && params.TransactionID == matchResult.Parameters.TransactionID {
			matchLogs = append(matchLogs, fmt.Sprintf(" +1 - Matched Transaction ID: %s", matchResult.Parameters.TransactionID))
			matchResult.Score++
		}

		if offerMatch.ClickID > 0 {
			// This is a direct match from the subquery, so we can give it a higher score
			matchResult.Score += 5
			matchLogs = append(matchLogs, fmt.Sprintf(" +5 - Matched Click ID: %s", strconv.FormatInt(offerMatch.ClickID, 10)))

			if click, err := s.Queries.GetRawClickByID(s.Ctx, offerMatch.ClickID); err == nil {
				matchResult.Click = &click

				clickURL, err := url.Parse(click.URL)
				if err != nil {
					log.Printf("Error parsing click URL %s: %v", click.URL, err)
					continue
				}

				if offerMatchURL.Hostname() == clickURL.Hostname() {
					matchResult.Score++
					matchLogs = append(matchLogs, fmt.Sprintf(" +1 - Matched on Hostname Click ID: %s", click.TransactionID))
				}

				if offerMatchURL.EscapedPath() == clickURL.EscapedPath() {
					matchResult.Score++
					matchLogs = append(matchLogs, fmt.Sprintf(" +1 - Matched on Path Click ID: %s", click.TransactionID))
				}
			}
		} else if matchResult.Parameters.TransactionID != "" {
			if click, err := s.Queries.GetRawClickByNetworkAndTransactionID(s.Ctx, GetRawClickByNetworkAndTransactionIDParams{
				NetworkID:     offerMatch.Offer.NetworkID,
				TransactionID: matchResult.Parameters.TransactionID,
			}); err == nil {
				// This didn't join directly on the subquery, but we found a matching click for the given Transaction ID
				matchResult.Score += 1
				matchLogs = append(matchLogs, fmt.Sprintf(" +1 - Matched Click ID via Transaction ID in params: %s", click.TransactionID))
				matchResult.Click = &click

				if click.OfferID == offerMatch.Offer.ID {
					matchResult.Score++
					matchLogs = append(matchLogs, fmt.Sprintf(" +1 - Matched OfferID to Click ID: %s", click.TransactionID))
				}

				if click.OfferURLID.Valid && offerMatch.OfferURL.ID == click.OfferURLID.Int64 {
					matchResult.Score++
					matchLogs = append(matchLogs, fmt.Sprintf(" +1 - Matched OfferURLID to Click ID: %s", click.TransactionID))
				}

				clickURL, err := url.Parse(click.URL)
				if err != nil {
					log.Printf("Error parsing click URL %s: %v", click.URL, err)
					continue
				}

				if offerMatchURL.Hostname() == clickURL.Hostname() {
					matchResult.Score++
					matchLogs = append(matchLogs, fmt.Sprintf(" +1 - Matched on Hostname Click ID: %s", click.TransactionID))
				}

				if offerMatchURL.EscapedPath() == clickURL.EscapedPath() {
					matchResult.Score++
					matchLogs = append(matchLogs, fmt.Sprintf(" +1 - Matched on Path Click ID: %s", click.TransactionID))
				}
			}
		}

		bestMatches[index] = matchResult
		if matchResult.Score > bestMatches[bestMatchIndex].Score {
			bestMatchIndex = index
		}

		logs = append(logs, fmt.Sprintf(
			"\n-------------------\nPotential Offer: %d of %d - %s (ID: %d - %s %d) || %s (ID: %d - %s %d) || Score: %d\n* URL: %s\n",
			index+1,
			offerCount,
			offerMatch.Offer.Name,
			offerMatch.Offer.ID,
			enum.NetworkEnum(offerMatch.Offer.NetworkID),
			offerMatch.ExternalOfferID.Int64,
			offerMatch.OfferURL.Name,
			offerMatch.OfferURL.ID,
			enum.NetworkEnum(offerMatch.Offer.NetworkID),
			offerMatch.ExternalOfferURLID.Int64,
			matchResult.Score,
			offerMatch.OfferURL.DestinationURL,
		), strings.Join(matchLogs, "\n"))
	}

	bestMatch := bestMatches[bestMatchIndex]
	bestMatch.Logs = logs

	return bestMatch, nil
}

type ClickComparison struct {
	Click *RawClick
	URL   *url.URL
}

type OldFindBestOfferMatchParams struct {
	LandingPageURL string
	Cookies        map[string]string
	Parameters     AbandonedLeadTokenParams
}

type OfferMatchResult struct {
	Score      int
	Offer      *Offer
	OfferURL   *OfferURL
	Parameters *AbandonedLeadTokenParams
	Click      *RawClick
	Logs       []string
}

// Landing Page URL can contain placeholders like {offer_id}, {{offer_id}} and __OFFER_ID__ so we need to match them
var placeholderTokenTest = regexp.MustCompile(`^({.*}|{{.*}}|__.*__)`)

func (s *Store) OldFindBestOfferMatch(params OldFindBestOfferMatchParams, verbose ...bool) (OfferMatchResult, error) {
	var isVerbose bool = true
	if len(verbose) > 0 {
		isVerbose = verbose[0]
	}

	var bestMatch OfferMatchResult

	if params.LandingPageURL == "" {
		return bestMatch, fmt.Errorf("landing page URL is required")
	}

	landingPage, err := url.Parse(params.LandingPageURL)
	if err != nil {
		return bestMatch, fmt.Errorf("invalid landing page URL: %s", params.LandingPageURL)
	}

	parsedLandingPageURL := strings.Join([]string{
		landingPage.Scheme,
		"://",
		landingPage.Hostname(),
		landingPage.EscapedPath(),
	}, "")

	if isVerbose {
		log.Printf("\nWorking request URL: %s\nUsing base URL: %s", params.LandingPageURL, parsedLandingPageURL)
	}

	matchedOffers, err := s.Queries.GetOffersByPageURL(s.Ctx, parsedLandingPageURL)
	if err != nil {
		return bestMatch, fmt.Errorf("error fetching offers by page URL: %w", err)
	}

	offerCount := len(matchedOffers)

	var availableClicks []ClickComparison
	if params.Parameters.TransactionID != "" {
		// locate any clicks for the given NetworkID & Transaction ID combo
		for _, offer := range matchedOffers {
			offerNetworkID := offer.Offer.NetworkID
			var networkIDFound bool

			for _, clickData := range availableClicks {
				if clickData.Click != nil && clickData.Click.NetworkID == offerNetworkID {
					networkIDFound = true
					break
				}
			}

			if !networkIDFound {
				if click, err := s.Queries.GetRawClickByNetworkAndTransactionID(s.Ctx, GetRawClickByNetworkAndTransactionIDParams{
					NetworkID:     offerNetworkID,
					TransactionID: params.Parameters.TransactionID,
				}); err == nil {
					clickURL, err := url.Parse(click.URL)
					if err != nil {
						log.Printf("Error parsing click URL %s: %v", click.URL, err)
						continue
					}

					availableClicks = append(availableClicks, ClickComparison{
						Click: &click,
						URL:   clickURL,
					})
				}
			}
		}
	}

	if isVerbose {
		fmt.Printf("Found %d clicks for Transaction ID: %s\n", len(availableClicks), params.Parameters.TransactionID)
	}

	landingPageQuery := landingPage.Query()

	for index, currentOfferMatch := range matchedOffers {
		var matchLog []string
		var score int = 1
		var currentParameters AbandonedLeadTokenParams
		var currentClick *RawClick

		currentOfferUrl, err := url.Parse(currentOfferMatch.OfferURL.DestinationURL)
		if err != nil {
			return bestMatch, fmt.Errorf("invalid offer URL: %s", currentOfferMatch.OfferURL.DestinationURL)
		}

		currentOfferQuery, err := url.ParseQuery(currentOfferUrl.RawQuery)
		if err != nil {
			return bestMatch, fmt.Errorf("invalid offer URL query: %s", currentOfferUrl.RawQuery)
		}

		for _, clickComparison := range availableClicks {
			if clickComparison.Click == nil || clickComparison.URL == nil {
				continue
			}

			if currentOfferMatch.Offer.ID == clickComparison.Click.OfferID {
				matchLog = append(matchLog, fmt.Sprintf(" + Matched Offer for Click ID: %s", clickComparison.Click.TransactionID))
				score++

				if currentClick != nil {
					if isVerbose {
						log.Printf("Multiple clicks found for offer %d, using the latest one: %s", currentOfferMatch.Offer.ID, currentClick.TransactionID)
					}
				}
				currentClick = clickComparison.Click
				currentParameters.UpdateTransactionID(clickComparison.Click.TransactionID)
			}

			if currentOfferUrl.Hostname() == clickComparison.URL.Hostname() && currentOfferUrl.EscapedPath() == clickComparison.URL.EscapedPath() {
				matchLog = append(matchLog, fmt.Sprintf(" + Matched on Hostname Click ID: %s", clickComparison.Click.TransactionID))
				score++

				if currentClick != nil {
					if isVerbose {
						log.Printf("Multiple clicks found for offer %d, using the latest one: %s", currentOfferMatch.Offer.ID, currentClick.TransactionID)
					}
				}
				currentClick = clickComparison.Click
				currentParameters.UpdateTransactionID(clickComparison.Click.TransactionID)
			}
		}

		if currentOfferUrl.Hostname() == landingPage.Hostname() {
			matchLog = append(matchLog, fmt.Sprintf(" + Matched hostname: %s", currentOfferUrl.Hostname()))
			score++
		}

		if currentOfferUrl.Path == landingPage.Path {
			matchLog = append(matchLog, fmt.Sprintf(" + Matched path: %s", currentOfferUrl.Path))
			score++
		}

		if currentOfferMatch.Offer.Status == int32(enum.OfferStatusActive) {
			matchLog = append(matchLog, " + Offer is active")
			score++
		}

		if currentOfferMatch.OfferURL.Status == int32(enum.OfferURLStatusActive) {
			matchLog = append(matchLog, " + Offer URL is active")
			score++
		}

		// Compare the query parameters of the offer with the landing page query parameters
		for offerQueryKey, offerQueryValues := range currentOfferQuery {
			if landingPageQuery.Has(offerQueryKey) { // s1, ck, etc...
				landingPageQueryValues := landingPageQuery[offerQueryKey]
				for _, currentOfferQueryValue := range offerQueryValues {
					currentOfferQueryValue = strings.TrimSpace(currentOfferQueryValue)

					// check if using placeholders like {offer_id}, {{offer_id}} or __OFFER_ID__
					if placeholderTokenTest.MatchString(currentOfferQueryValue) {
						for _, currentLandingPageQueryValue := range landingPageQueryValues {
							currentLandingPageQueryValue = strings.TrimSpace(currentLandingPageQueryValue)
							if currentLandingPageQueryValue != "" {
								isNetworkMatch := currentOfferMatch.Offer.CheckNetworkMatch(&currentLandingPageQueryValue)

								if strings.Contains(currentOfferQueryValue, "transaction_id") || strings.Contains(currentOfferQueryValue, "transaction.id") || strings.Contains(currentOfferQueryValue, "CLICKID") {
									currentParameters.UpdateTransactionID(currentLandingPageQueryValue)
								} else if strings.Contains(currentOfferQueryValue, "affiliate_id") || strings.Contains(currentOfferQueryValue, "affiliate.id") {
									currentParameters.AffiliateID = currentLandingPageQueryValue
								} else if strings.Contains(currentOfferQueryValue, "campaign_id") || strings.Contains(currentOfferQueryValue, "campaign.id") {
									currentParameters.CampaignID = currentLandingPageQueryValue
								} else if strings.Contains(currentOfferQueryValue, "offer_id") {
									currentParameters.OfferID = currentLandingPageQueryValue
								}

								if isNetworkMatch {
									matchLog = append(matchLog, fmt.Sprintf(" + Matched placeholder query parameter + network %s: %s = %s", currentOfferQueryValue, offerQueryKey, currentLandingPageQueryValue))
									score++
								} else {
									matchLog = append(matchLog, fmt.Sprintf(" + Matched placeholder query parameter %s: %s = %s", currentOfferQueryValue, offerQueryKey, currentLandingPageQueryValue))
								}
								score++

								break
							}
						}
					} else {
						if slices.Contains(landingPageQueryValues, currentOfferQueryValue) {
							isNetworkMatch := currentOfferMatch.Offer.CheckNetworkMatch(&currentOfferQueryValue)

							switch offerQueryKey {
							case "transaction_id", "_ef_transaction_id":
								currentParameters.UpdateTransactionID(currentOfferQueryValue)
							case "ef_oid", "oid", "offer_id":
								currentParameters.OfferID = currentOfferQueryValue
							case "uid", "ef_uid", "offer_url_id":
								currentParameters.OfferURLID = currentOfferQueryValue
							case "affid", "aid", "affiliate_id":
								currentParameters.AffiliateID = currentOfferQueryValue
							}

							if isNetworkMatch {
								matchLog = append(matchLog, fmt.Sprintf(" + Matched query parameter + network %s: %s = %s", offerQueryKey, currentOfferQueryValue, currentOfferQueryValue))
								score++
							} else {
								matchLog = append(matchLog, fmt.Sprintf(" + Matched query parameter: %s = %s", offerQueryKey, currentOfferQueryValue))
							}
							score++
						}
					}
				}
			}
		}

		if currentParameters.CompareOffer(currentOfferMatch.Offer, currentOfferMatch.ExternalOfferID) {
			matchLog = append(matchLog, fmt.Sprintf(" + Matched Offer ID: %s", currentParameters.OfferID))
			score++
		}

		if currentParameters.CompareOfferURL(currentOfferMatch.OfferURL, currentOfferMatch.ExternalOfferURLID) {
			matchLog = append(matchLog, fmt.Sprintf(" + Matched Offer URL ID: %s", currentParameters.OfferURLID))
			score++
		}

		if currentParameters.TransactionID != "" && params.Parameters.TransactionID == currentParameters.TransactionID {
			matchLog = append(matchLog, fmt.Sprintf(" + Matched Transaction ID: %s", currentParameters.TransactionID))
			score++
		}

		if isVerbose {
			fmt.Printf(
				"-- Potential Offer: %d of %d - (ID: %d - %s ID: %d) %s || %s (ID: %d) || score: %d\n * URL: %s\n",
				index+1,
				offerCount,
				currentOfferMatch.Offer.ID,
				enum.NetworkEnum(currentOfferMatch.Offer.NetworkID),
				currentOfferMatch.ExternalOfferID.Int64,
				currentOfferMatch.Offer.Name,
				currentOfferMatch.OfferURL.Name,
				currentOfferMatch.ExternalOfferURLID.Int64,
				score,
				currentOfferMatch.OfferURL.DestinationURL,
			)

			if len(matchLog) > 0 {
				fmt.Println(strings.Join(matchLog, "\n"))
			}
		}

		if score > bestMatch.Score {
			bestMatch.Score = score
			bestMatch.Offer = &currentOfferMatch.Offer
			bestMatch.OfferURL = &currentOfferMatch.OfferURL
			bestMatch.Parameters = &currentParameters
			bestMatch.Click = currentClick
		}
	}

	if bestMatch.Score == 0 {
		return bestMatch, fmt.Errorf("no matching offer found for landing page URL: %s", params.LandingPageURL)
	}

	// We matched an offer, now we update the Parameters to account for any URL parameters
	if bestMatch.Parameters != nil {
		bestMatch.Parameters.MergeOtherParams(params.Parameters)
	}

	return bestMatch, nil
}

// CheckNetworkMatch reports whether a numeric landing-page value marked with the
// secondary-network suffix belongs to this offer. The suffix is "_secondary".
func (o *Offer) CheckNetworkMatch(value *string) bool {
	if o != nil && value != nil && strings.HasSuffix(*value, "_secondary") {
		trimmedValue := strings.TrimSuffix(*value, "_secondary")

		if _, err := strconv.Atoi(trimmedValue); err == nil {
			// we have a numeric value, so this is most likely an ID
			*value = trimmedValue

			if o.NetworkID != int32(enum.NetworkSecondary) {
				return false
			}
		}
	}

	return true
}
