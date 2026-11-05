package requests

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/nickarnoldi/funnel-service/internal/db"
	"github.com/nickarnoldi/funnel-service/internal/enum"
	"github.com/nickarnoldi/funnel-service/internal/utils"
)

type AbandonedLeadRequest struct {
	LandingPageURL string            `json:"landing_page"`
	SessionLength  int32             `json:"session_length"`
	StepNumber     int32             `json:"step_number"`
	IPAddress      string            `json:"ip_address"`
	Referrer       string            `json:"referrer"`
	Cookies        map[string]string `json:"cookies"`
	Body           map[string]any    `json:"body"`
}

func (request *AbandonedLeadRequest) Process(store *db.Store) (AbandonedLeadResult, error) {
	if !request.IsValid() {
		return AbandonedLeadResult{}, fmt.Errorf("bad request")
	}

	matchResult, err := request.FindBestMatches(store, false)

	offer := matchResult.Offer
	offerURL := matchResult.OfferURL
	parameters := matchResult.Parameters

	if err != nil || offer == nil || offerURL == nil || parameters == nil {
		log.Printf("Error finding best offer by landing page: %v", err)
		return AbandonedLeadResult{}, fmt.Errorf("not found")
	}

	journeyResult := request.ProcessFunnelJourney(store, matchResult, false)

	body, err := json.Marshal(request.Body)
	if err != nil {
		log.Printf("Error marshalling request body: %v", err)
	}

	cookies, err := json.Marshal(request.Cookies)
	if err != nil {
		log.Printf("Error marshalling request cookies: %v", err)
	}

	var lead db.AbandonedLead = db.AbandonedLead{
		OfferID:        offer.ID,
		OfferURLID:     offerURL.ID,
		StepNumber:     journeyResult.StepNumber,
		TransactionID:  parameters.TransactionID,
		AffiliateID:    db.String(&parameters.AffiliateID),
		CampaignID:     db.String(&parameters.CampaignID),
		HoneypotFilled: journeyResult.HoneypotFilled,
		SessionLength:  request.SessionLength,
		Body:           body,
		Cookies:        cookies,
	}

	lead.LandingPageURL = request.LandingPageURL
	lead.IPAddress = request.IPAddress
	lead.Referrer = request.Referrer

	if matchResult.Click != nil {
		lead.ClickID = db.BigInt(&matchResult.Click.ID)

		var geolocation db.Geolocation
		if err := json.Unmarshal(matchResult.Click.Geolocation, &geolocation); err != nil {
			log.Printf("Error unmarshalling geolocation: %v", err)
		}

		geolocation.Normalize()

		lead.City = db.String(&geolocation.CityName)
		lead.PostalCode = db.String(&geolocation.PostalCode)
		lead.CountryCode = db.String(&geolocation.CountryCode)
		lead.RegionCode = db.String(&geolocation.RegionCode)

		var deviceInfo db.DeviceInfo
		if err := json.Unmarshal(matchResult.Click.DeviceInfo, &deviceInfo); err != nil {
			log.Printf("Error unmarshalling device info: %v", err)
		}

		lead.IsBot = deviceInfo.IsRobot
		lead.IsMobile = deviceInfo.IsMobile
	} else {
		// No base click, use what we have to fill in the gaps

		// todo geoip lookup based on IP address
	}

	var dob, dobDay, dobMonth, dobYear string
	request.ExtractBodyValue(&dobDay, "dob_day", "birth_day", "date_of_birth_day", "dateOfBirthDay")
	request.ExtractBodyValue(&dobMonth, "dob_month", "birth_month", "date_of_birth_month", "dateOfBirthMonth")
	request.ExtractBodyValue(&dobYear, "dob_year", "birth_year", "date_of_birth_year", "dateOfBirthYear")
	request.ExtractBodyValue(&dob, "date_of_birth", "dob", "birth_date", "dateOfBirth")

	if dobDay != "" && dobMonth != "" && dobYear != "" {
		birthDate, err := time.Parse("2006-01-02", fmt.Sprintf("%s-%02s-%02s", dobYear, dobMonth, dobDay))
		if err == nil {
			lead.DateOfBirth.Scan(birthDate)
		}
	}

	if dob != "" && !lead.DateOfBirth.Valid {
		birthDate, err := time.Parse("2006-01-02", dob)
		if err == nil {
			lead.DateOfBirth.Scan(birthDate)
		} else {
			// try alt format
			birthDate, err = time.Parse("01/02/2006", dob)
			if err == nil {
				lead.DateOfBirth.Scan(birthDate)
			} else {
				birthDate, err = time.Parse("1/2/2006", dob)
				if err == nil {
					lead.DateOfBirth.Scan(birthDate)
				}
			}
		}
	}

	var phoneNumber string
	request.ExtractBodyValue(&phoneNumber, "phone", "phone_number", "phoneNumber", "phone_cell", "phone_home")
	if phoneNumber != "" {
		phoneNumber = nonDigitRegex.ReplaceAllString(phoneNumber, "")
		if len(phoneNumber) > 20 {
			phoneNumber = phoneNumber[:20]
		}
		lead.PhoneNumber.Scan(phoneNumber)
	}

	var email string
	request.ExtractBodyValue(&email, "email", "email_address", "emailAddress")
	if email != "" {
		if len(email) > 100 {
			email = email[:100]
		}
		lead.Email.Scan(email)
	}

	var jornayaID string
	request.ExtractBodyValue(&jornayaID, "universal_leadid", "jornaya", "jornaya_lead_id", "jornaya_id")
	if jornayaID != "" {
		if len(jornayaID) > 45 {
			jornayaID = jornayaID[:45]
		}
		lead.JornayaID.Scan(jornayaID)
	}

	var anuraScore string
	request.ExtractBodyValue(&anuraScore, "anura_score", "anura")
	if anuraScore != "" {
		if len(anuraScore) > 15 {
			anuraScore = anuraScore[:15]
		}
		lead.AnuraScore.Scan(anuraScore)
	}

	var monthlyIncome string
	request.ExtractBodyValue(&monthlyIncome, "monthly_income", "gross_monthly_income")
	if monthlyIncome == "" {
		monthlyIncome = "0"
	} else if len(monthlyIncome) > 10 {
		monthlyIncome = monthlyIncome[:10]
	}
	lead.GrossMonthlyIncome.Scan(monthlyIncome)

	var employmentStatusString string
	request.ExtractBodyValue(&employmentStatusString, "employment_status", "employmentStatus")
	var employmentStatus enum.EmploymentStatusEnum
	employmentStatus.FromString(employmentStatusString)
	employmentStatusInt16 := int16(employmentStatus)
	lead.EmploymentStatus = db.SmallInt(&employmentStatusInt16)

	var trustedFormURL string
	request.ExtractBodyValue(&trustedFormURL, "xxTrustedFormToken", "xxTrustedFormCertUrl")
	var trustedFormPingURL string
	request.ExtractBodyValue(&trustedFormPingURL, "xxTrustedFormPingUrl")
	var trustedFormID string
	if trustedFormURL != "" {
		trustedFormID = strings.TrimPrefix(trustedFormURL, "https://cert.trustedform.com/")
		if len(trustedFormID) > 45 {
			trustedFormID = trustedFormID[:45]
		}
		lead.TrustedformID = db.String(&trustedFormID)
	}

	var country string
	request.ExtractBodyValue(&country, "country", "country_code")
	if country != "" && len(country) == 2 {
		country = strings.ToUpper(country)
		if len(country) > 5 {
			country = country[:5]
		}
		lead.CountryCode = db.String(&country)
	}

	var region string
	request.ExtractBodyValue(&region, "region", "region_code")
	if region != "" && len(region) == 2 {
		region = strings.ToUpper(region)
		if len(region) > 5 {
			region = region[:5]
		}
		lead.RegionCode = db.String(&region)
	}

	var city string
	request.ExtractBodyValue(&city, "city", "city_name")
	if city != "" {
		city = utils.ToUppercaseWords(strings.TrimSpace(city))
		if len(city) > 100 {
			city = city[:100]
		}
		lead.City = db.String(&city)
	}

	var postalCode string
	request.ExtractBodyValue(&postalCode, "zip", "zip_code", "zipcode", "postal_code", "postalcode", "postalzip", "postcode", "post_code")
	if postalCode != "" {
		postalCode = strings.ToUpper(utils.RemoveAllWhitespace(postalCode))
		if len(postalCode) > 25 {
			postalCode = postalCode[:25]
		}
		lead.PostalCode = db.String(&postalCode)
	}

	return AbandonedLeadResult{
		AbandonedLead: &lead,
		Step:          journeyResult.CurrentStep,
	}, nil
}

// ProcessFunnelJourney processes the funnel journey for the abandoned lead request.
// It determines the current step based on the funnel conditions and fields, and obfuscates SSN/SIN-like values in the payload body.
// It returns the step number, the current step, and whether a honeypot field was filled.
func (request *AbandonedLeadRequest) ProcessFunnelJourney(store *db.Store, offerMatch db.OfferMatchResult, verbose ...bool) FunnelJourneyResult {
	var isVerbose bool = true
	if len(verbose) > 0 {
		isVerbose = verbose[0]
	}

	var obfuscated bool
	var doNotSave bool

	stepNumber := int32(1)
	var currentStep *FunnelJourney
	var honeypotFilled bool

	if request.StepNumber != 0 {
		stepNumber = request.StepNumber
	}

	// Iterate over the payload and obfuscate SSN/SIN-like values
	for key, value := range request.Body {
		var stringifiedValue string
		switch value := value.(type) {
		case string:
			stringifiedValue = strings.TrimSpace(value)
		case int, int32, int64:
			stringifiedValue = fmt.Sprintf("%d", value)
		}

		if key == "sin" || strings.Contains(key, "sin") || key == "ssn" || strings.Contains(key, "ssn") {
			if ssnOrSinRegex.MatchString(stringifiedValue) {
				// First 3, last 3
				obfuscatedValue := fmt.Sprintf("%s***%s", stringifiedValue[:3], stringifiedValue[len(stringifiedValue)-3:])
				fmt.Printf("Obfuscating: Found SSN/SIN-like value in key `%s`: %s\n", key, obfuscatedValue)
				request.Body[key] = obfuscatedValue
				obfuscated = true
			}
		}
	}

	// If the offer URL has a funnel ID, we need to process the funnel journey on conditional forms
	// This will determine the current step based on the funnel conditions and fields
	if offerMatch.OfferURL.FunnelID.Valid {
		var funnelJourney = make(map[int32]FunnelJourney)
		var funnelStepsFieldsMap = make(map[int64][]*db.FunnelField)
		var funnelStepsConditionsMap = make(map[int64][]*db.FunnelStepCondition)

		funnelFields, _ := store.Queries.GetFunnelFieldsByFunnelID(store.Ctx, offerMatch.OfferURL.FunnelID.Int64)
		for _, field := range funnelFields {
			if field.FunnelStepID.Valid {
				funnelStepsFieldsMap[field.FunnelStepID.Int64] = append(funnelStepsFieldsMap[field.FunnelStepID.Int64], &field)
			}

			fieldValue, fieldExists := request.Body[field.Name]
			if fieldExists {
				var stringifiedFieldValue string
				switch fieldValue := fieldValue.(type) {
				case string:
					stringifiedFieldValue = strings.TrimSpace(fieldValue)
				case int, int32, int64:
					stringifiedFieldValue = fmt.Sprintf("%d", fieldValue)
				}

				if field.IsHoneypot {
					if stringifiedFieldValue != "" {
						honeypotFilled = true
					}
				}
			}

			if field.DoNotSave {
				delete(request.Body, field.Name)
				doNotSave = true
			}
		}

		funnelSteps, err := store.Queries.GetFunnelStepsByFunnelID(store.Ctx, offerMatch.OfferURL.FunnelID.Int64)
		if err == nil {
			funnelConditions, _ := store.Queries.GetFunnelStepConditionsByFunnelID(store.Ctx, offerMatch.OfferURL.FunnelID.Int64)
			for _, condition := range funnelConditions {
				funnelStepsConditionsMap[condition.FunnelStepID] = append(funnelStepsConditionsMap[condition.FunnelStepID], &condition)
			}

			funnelStepCount := len(funnelSteps)

			for _, step := range funnelSteps {
				if _, check := funnelJourney[step.StepIndex]; check {
					// Discard since we already have a step at this index
					fmt.Printf("Funnel step %d (%s) has step index %d, but that index already exists in funnel journey\n", step.ID, step.Name, step.StepIndex)
					funnelStepCount--
				} else {
					funnelJourney[step.StepIndex] = FunnelJourney{
						FunnelStep: step,
						Fields:     funnelStepsFieldsMap[step.ID],
						Conditions: funnelStepsConditionsMap[step.ID],
					}
				}
			}

			// Process the funnel journey to determine the current step
			var stepCounter int32
			for stepIndex := int32(0); stepIndex < int32(len(funnelJourney)); stepIndex++ {
				step, exists := funnelJourney[stepIndex]
				if !exists {
					continue
				}
				currentStep = &step

				var skipStep bool

				if isVerbose {
					fmt.Printf("Funnel Step %d: %s (ID: %d)\n", stepCounter, step.Name, step.ID)
				}

				for _, condition := range step.Conditions {
					fieldValue, found := request.Body[condition.Field]
					if condition.Field != "" && found {
						switch fieldValue := fieldValue.(type) {
						case string:
							// @todo: handle other operators, if needed
							if condition.Operator == int32(enum.ConditionOperatorEqual) && fieldValue == condition.Condition.String {
								skipStep = true
							} else if condition.Operator == int32(enum.ConditionOperatorNotEqual) && fieldValue != condition.Condition.String {
								skipStep = true
							}
						}

						if skipStep {
							if isVerbose {
								fmt.Printf(
									"Condition met: %s %s %s. Skipping Step# %d: %s\n",
									condition.Field,
									enum.ConditionOperator[enum.ConditionOperatorEnum(condition.Operator)],
									condition.Condition.String,
									stepIndex+1,
									funnelJourney[stepIndex].Name,
								)
							}
							break
						}
					}
				}

				if skipStep {
					continue
				}

				stepCounter++
				if stepCounter >= stepNumber {
					break
				}
			}
		}
	}

	return FunnelJourneyResult{
		StepNumber:     stepNumber,
		CurrentStep:    currentStep,
		HoneypotFilled: honeypotFilled,
		Obfuscated:     obfuscated,
		DoNotSave:      doNotSave,
	}
}

func (request *AbandonedLeadRequest) FindBestMatches(store *db.Store, verbose ...bool) (db.OfferMatchResult, error) {
	parameters := request.GetLeadParameters()

	return store.FindBestOfferMatch(db.FindBestOfferMatchParams{
		LandingPageURL: request.LandingPageURL,
		TransactionID:  parameters.TransactionID,
		OfferID:        parameters.OfferID,
		OfferURLID:     parameters.OfferURLID,
	})
}

func (request *AbandonedLeadRequest) GetLeadParameters() db.AbandonedLeadTokenParams {
	var parameters db.AbandonedLeadTokenParams

	for key, value := range request.Cookies {
		if !strings.Contains(value, " ") && !strings.Contains(value, "-") {
			switch key {
			case "transaction_id", "_ef_transaction_id", "ef_transaction_id", "transaction.id":
				parameters.UpdateTransactionID(value)
			case "s1", "s2", "s3", "s4", "s5":
				if parameters.TransactionID == "" {
					parameters.UpdateTransactionID(value)
				}
			case "uid", "ef_uid":
				parameters.OfferURLID = value
			case "oid", "ef_oid":
				parameters.OfferID = value
			case "affid", "aid", "ef_affid":
				parameters.AffiliateID = value
			case "cpid", "ci", "campaign_id", "campaignid":
				parameters.CampaignID = value
			}
		}

		if strings.HasPrefix(key, "ef_tid_") {
			if strings.Contains(value, "|") {
				parts := strings.Split(value, "|")
				value = parts[len(parts)-1] // Take the last part after the pipe
			}

			if value != "" {
				parameters.UpdateTransactionID(value)
			}
		}
	}

	var transactionID string
	request.ExtractBodyValue(&transactionID, "etid", "transaction_id")
	if transactionID != "" {
		parameters.UpdateTransactionID(transactionID)
	}

	return parameters
}

func (request *AbandonedLeadRequest) IsValid() bool {
	if request.LandingPageURL == "" {
		log.Printf("LandingPageURL is empty")
		return false
	}

	if request.IPAddress == "" {
		log.Printf("IPAddress is empty")
		return false
	}

	if request.Referrer == "" {
		log.Printf("Referrer is empty")
		return false
	}

	return true
}

func (request *AbandonedLeadRequest) ExtractBodyValue(returnValue any, possibleKeys ...string) bool {
	// nil pointer check
	if returnValue == nil {
		return false
	}

	for _, key := range possibleKeys {
		bodyValue, ok := request.Body[key]
		if !ok {
			continue
		}

		// payload.Body[key] can be of various types, so we need to handle them accordingly
		switch typedBodyValue := bodyValue.(type) {
		case string:
			switch typedReturnValue := any(returnValue).(type) {
			case *string:
				if len(typedBodyValue) == 0 {
					continue
				}

				*typedReturnValue = strings.TrimSpace(typedBodyValue)
				return true
			case *int64:
				intValue, err := strconv.ParseInt(typedBodyValue, 10, 64)
				if err != nil {
					continue
				}

				*typedReturnValue = intValue
				return true
			case *int32:
				intValue, err := strconv.ParseInt(typedBodyValue, 10, 32)
				if err != nil {
					continue
				}

				*typedReturnValue = int32(intValue)
				return true
			case *int:
				intValue, err := strconv.ParseInt(typedBodyValue, 10, 32)
				if err != nil {
					continue
				}

				*typedReturnValue = int(intValue)
				return true
			default:
				continue
			}
		case map[string]any:
			switch typedReturnValue := any(returnValue).(type) {
			case *map[string]any:
				if typedBodyValue == nil {
					continue
				}

				*typedReturnValue = any(typedBodyValue).(map[string]any)
				return true
			default:
				continue
			}
		case int, int32, int64, float32, float64:
			switch typedReturnValue := any(returnValue).(type) {
			case *string:
				*typedReturnValue = strconv.FormatFloat(typedBodyValue.(float64), 'f', -1, 64)
				return true
			case *int64:
				*typedReturnValue = int64(typedBodyValue.(float64))
				return true
			case *int32:
				*typedReturnValue = int32(typedBodyValue.(float64))
				return true
			case *int:
				*typedReturnValue = int(typedBodyValue.(float64))
				return true
			}
		default:
			continue
		}
	}

	return false
}

var ssnOrSinRegex = regexp.MustCompile(`(\d{3})-?(\d{2})-?(\d{4})|(\d{3})-?(\d{3})-?(\d{3})`)
var nonDigitRegex = regexp.MustCompile(`\D`)

type FunnelJourneyResult struct {
	StepNumber     int32
	CurrentStep    *FunnelJourney
	HoneypotFilled bool
	Obfuscated     bool
	DoNotSave      bool
}

type AbandonedLeadResult struct {
	AbandonedLead *db.AbandonedLead
	Step          *FunnelJourney
}

type FunnelJourney struct {
	db.FunnelStep
	Fields     []*db.FunnelField
	Conditions []*db.FunnelStepCondition
}

type CompareMatchResult struct {
	OfferMatch db.OfferMatchResult
	TestMatch  db.OfferMatchResult
	Score      int
	Logs       []string
}

/**
 * TestBestMatches tests the best offer match logic against the existing FindBestOfferMatch method.
 * It returns a CompareMatchResult containing both the test match and the existing match, along with a score and logs.
 */
func (request *AbandonedLeadRequest) TestBestMatches(store *db.Store, verbose ...bool) (CompareMatchResult, error) {
	parameters := request.GetLeadParameters()

	testMatch, err := store.FindBestOfferMatch(db.FindBestOfferMatchParams{
		LandingPageURL: request.LandingPageURL,
		TransactionID:  parameters.TransactionID,
		OfferID:        parameters.OfferID,
		OfferURLID:     parameters.OfferURLID,
	})
	if err != nil {
		fmt.Printf("Error in test matching: %v\n", err)
	}

	match, err := store.OldFindBestOfferMatch(db.OldFindBestOfferMatchParams{
		LandingPageURL: request.LandingPageURL,
		Cookies:        request.Cookies,
		Parameters:     parameters,
	}, verbose...)

	if err == nil && testMatch.Score < match.Score {
		fmt.Printf("Warning: Test match score (%d) is less than actual match score (%d)\n", testMatch.Score, match.Score)

		for _, logEntry := range testMatch.Logs {
			fmt.Println(logEntry)
		}

		os.Exit(0)
	}

	var logs []string = make([]string, 0)
	var score int
	if err == nil && testMatch.Offer != nil {
		if match.Offer != nil {
			if match.Offer.ID != testMatch.Offer.ID {
				score--
				logs = append(logs, fmt.Sprintf("Mismatch in best offer match: TestMatch Offer ID %d vs Match Offer ID %d\n", testMatch.Offer.ID, match.Offer.ID))
			} else {
				score++
			}
		}

		if match.OfferURL != nil {
			if match.OfferURL.ID != testMatch.OfferURL.ID {
				score--
				logs = append(logs, fmt.Sprintf("Mismatch in best offer URL match: TestMatch OfferURL ID %d vs Match OfferURL ID %d\n", testMatch.OfferURL.ID, match.OfferURL.ID))
				logs = append(logs, fmt.Sprintf("Base: %s\nMatch: %s\nTest: %s\n", request.LandingPageURL, match.OfferURL.DestinationURL, testMatch.OfferURL.DestinationURL))
			} else {
				score++
			}
		}

		if testMatch.Click == nil && match.Click != nil {
			score--
			logs = append(logs, fmt.Sprintf("Mismatch in best ClickID match: Test Click is nil vs Match ClickID: %s\n", match.Click.TransactionID))
		} else if testMatch.Click != nil && match.Click == nil && parameters.TransactionID != "" {
			score++
			logs = append(logs, fmt.Sprintf("Mismatch in best ClickID match: Test ClickID: %s vs Match Click is nil\n", testMatch.Click.TransactionID))
		} else if testMatch.Click != nil && match.Click != nil && testMatch.Click.ID != match.Click.ID {
			score--
			logs = append(logs, fmt.Sprintf("Mismatch in best ClickID match: Test ClickID: %s vs Match ClickID: %s\n", testMatch.Click.TransactionID, match.Click.TransactionID))
		} else {
			score++
		}
	}

	return CompareMatchResult{
		TestMatch:  testMatch,
		OfferMatch: match,
		Score:      score,
		Logs:       logs,
	}, err
}
