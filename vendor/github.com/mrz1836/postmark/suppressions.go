package postmark

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// SuppressionReasonType - The reason type of suppression
type SuppressionReasonType string

// OriginType - The reason type of origin
type OriginType string

const (
	// HardBounceReason means an email sent to the address returned a hard bounce.
	HardBounceReason SuppressionReasonType = "HardBounce"

	// SpamComplaintReason means the recipient marked an email as spam.
	SpamComplaintReason SuppressionReasonType = "SpamComplaint"

	// ManualSuppressionReason means the recipient followed an unsubscribe link.
	ManualSuppressionReason SuppressionReasonType = "ManualSuppression"

	// RecipientOrigin means the email was added to the suppression list
	// as a result of the recipient's own action, e.g. by following an unsubscribe link.
	RecipientOrigin OriginType = "Recipient"

	// CustomerOrigin means the email was added to the suppression list as
	// the result of action by the Postmark account holder (e.g. Postmark's
	// customer).
	CustomerOrigin OriginType = "Customer"

	// AdminOrigin means the email was added to the suppression list as
	// the result of action by Postmark staff.
	AdminOrigin OriginType = "Admin"
)

// Suppression contains a suppressed email address for a particular message stream.
type Suppression struct {
	// EmailAddress is the address that is suppressed (can't be emailed any more)
	EmailAddress string

	// SuppressionReason is why the email address was added to the suppression list.
	// Possible options: HardBounce, SpamComplaint, ManualSuppression
	SuppressionReason SuppressionReasonType

	// Origin describes who added the email address to the suppression list.
	// Possible options: Recipient, Customer, Admin.
	Origin OriginType

	// CreatedAt is when the email address was added to the suppression list.
	CreatedAt time.Time
}

// suppressionsResponse - A message received from the Postmark server
type suppressionsResponse struct {
	// Suppressions - The slice of suppression email address.
	Suppressions []Suppression
}

// GetSuppressions fetches email addresses in the list of suppression dump on the server
// It returns a Suppressions slice, and any error that occurred
// https://postmarkapp.com/developer/api/suppressions-api#suppression-dump
func (client *Client) GetSuppressions(
	ctx context.Context,
	streamID string,
	options map[string]interface{},
) ([]Suppression, error) {

	values := &url.Values{}
	for k, v := range options {
		values.Add(k, fmt.Sprintf("%v", v))
	}

	path := fmt.Sprintf("message-streams/%s/suppressions/dump", streamID)
	if len(options) != 0 {
		path = fmt.Sprintf("%s?%s", path, values.Encode())
	}

	res := suppressionsResponse{}
	err := client.doRequest(ctx, parameters{
		Method:    http.MethodGet,
		Path:      path,
		TokenType: serverToken,
	}, &res)
	return res.Suppressions, err
}
