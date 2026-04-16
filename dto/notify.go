package dto

type Notify struct {
	Type    string        `json:"type"`
	Title   string        `json:"title"`
	Content string        `json:"content"`
	Values  []interface{} `json:"values"`
}

const ContentValueParam = "{{value}}"

const (
	NotifyTypeQuotaExceed   = "quota_exceed"
	NotifyTypeChannelUpdate = "channel_update"
	NotifyTypeChannelTest   = "channel_test"

	NotifyTypeDistributorApplicationApproved = "distributor_application_approved"
	NotifyTypeDistributorApplicationRejected = "distributor_application_rejected"
	NotifyTypeDistributorRoleGranted         = "distributor_role_granted"
	NotifyTypeDistributorRoleRevoked         = "distributor_role_revoked"
	NotifyTypeDistributorWithdrawalSubmitted = "distributor_withdrawal_submitted"
	NotifyTypeDistributorWithdrawalApproved  = "distributor_withdrawal_approved"
	NotifyTypeDistributorWithdrawalRejected  = "distributor_withdrawal_rejected"

	NotifyTypeUserDemotedFromAdmin = "user_demoted_from_admin"
)

func NewNotify(t string, title string, content string, values []interface{}) Notify {
	return Notify{
		Type:    t,
		Title:   title,
		Content: content,
		Values:  values,
	}
}
