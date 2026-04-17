package main

import (
	"fmt"
	"regexp"
)

const (
	maxFieldValueLen = 1024
	maxAdSlots       = 50
)

var (
	placementIDPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,128}$`)
	campaignIDPattern  = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,64}$`)
)

// validateBidderRequest checks that a bid request contains valid, safe data.
func validateBidderRequest(req map[string]interface{}) error {
	if req == nil {
		return fmt.Errorf("request body is required")
	}

	// Enforce a cap on total number of keys to prevent hash flooding
	if len(req) > 20 {
		return fmt.Errorf("request contains too many fields (max 20, got %d)", len(req))
	}

	// Validate placement_id if present
	if pid, ok := req["placement_id"]; ok {
		s, ok := pid.(string)
		if !ok {
			return fmt.Errorf("placement_id must be a string")
		}
		if !placementIDPattern.MatchString(s) {
			return fmt.Errorf("placement_id contains invalid characters or exceeds length")
		}
	}

	// Validate campaign_id if present
	if cid, ok := req["campaign_id"]; ok {
		s, ok := cid.(string)
		if !ok {
			return fmt.Errorf("campaign_id must be a string")
		}
		if !campaignIDPattern.MatchString(s) {
			return fmt.Errorf("campaign_id contains invalid characters or exceeds length")
		}
	}

	// Validate ad_slots count if present
	if slots, ok := req["ad_slots"]; ok {
		slice, ok := slots.([]interface{})
		if !ok {
			return fmt.Errorf("ad_slots must be an array")
		}
		if len(slice) > maxAdSlots {
			return fmt.Errorf("ad_slots exceeds maximum count (max %d, got %d)", maxAdSlots, len(slice))
		}
	}

	// Check all string values for excessive length (prevents log injection, memory abuse)
	for key, val := range req {
		if s, ok := val.(string); ok {
			if len(s) > maxFieldValueLen {
				return fmt.Errorf("field %q exceeds maximum length (%d)", key, maxFieldValueLen)
			}
			// Reject control characters (prevents log injection)
			for _, c := range s {
				if c < 0x20 && c != '\t' {
					return fmt.Errorf("field %q contains invalid control characters", key)
				}
			}
		}
	}

	return nil
}
