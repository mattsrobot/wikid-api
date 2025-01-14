// Code generated by github.com/Khan/genqlient, DO NOT EDIT.

package railway

import (
	"time"

	"github.com/Khan/genqlient/graphql"
)

type Builder string

const (
	BuilderHeroku   Builder = "HEROKU"
	BuilderNixpacks Builder = "NIXPACKS"
	BuilderPaketo   Builder = "PAKETO"
)

type CDNProvider string

const (
	CDNProviderDetectedCdnProviderCloudflare  CDNProvider = "DETECTED_CDN_PROVIDER_CLOUDFLARE"
	CDNProviderDetectedCdnProviderUnspecified CDNProvider = "DETECTED_CDN_PROVIDER_UNSPECIFIED"
	CDNProviderUnrecognized                   CDNProvider = "UNRECOGNIZED"
)

type CertificateStatus string

const (
	CertificateStatusCertificateStatusTypeIssueFailed CertificateStatus = "CERTIFICATE_STATUS_TYPE_ISSUE_FAILED"
	CertificateStatusCertificateStatusTypeIssuing     CertificateStatus = "CERTIFICATE_STATUS_TYPE_ISSUING"
	CertificateStatusCertificateStatusTypeUnspecified CertificateStatus = "CERTIFICATE_STATUS_TYPE_UNSPECIFIED"
	CertificateStatusCertificateStatusTypeValid       CertificateStatus = "CERTIFICATE_STATUS_TYPE_VALID"
	CertificateStatusUnrecognized                     CertificateStatus = "UNRECOGNIZED"
)

// CustomDomainCreateCustomDomainCreateCustomDomain includes the requested fields of the GraphQL type CustomDomain.
type CustomDomainCreateCustomDomainCreateCustomDomain struct {
	Id            string                                                  `json:"id"`
	Domain        string                                                  `json:"domain"`
	CreatedAt     *time.Time                                              `json:"createdAt"`
	UpdatedAt     *time.Time                                              `json:"updatedAt"`
	ServiceId     string                                                  `json:"serviceId"`
	EnvironmentId string                                                  `json:"environmentId"`
	Status        *CustomDomainCreateCustomDomainCreateCustomDomainStatus `json:"status"`
}

// GetId returns CustomDomainCreateCustomDomainCreateCustomDomain.Id, and is useful for accessing the field via an interface.
func (v *CustomDomainCreateCustomDomainCreateCustomDomain) GetId() string { return v.Id }

// GetDomain returns CustomDomainCreateCustomDomainCreateCustomDomain.Domain, and is useful for accessing the field via an interface.
func (v *CustomDomainCreateCustomDomainCreateCustomDomain) GetDomain() string { return v.Domain }

// GetCreatedAt returns CustomDomainCreateCustomDomainCreateCustomDomain.CreatedAt, and is useful for accessing the field via an interface.
func (v *CustomDomainCreateCustomDomainCreateCustomDomain) GetCreatedAt() *time.Time {
	return v.CreatedAt
}

// GetUpdatedAt returns CustomDomainCreateCustomDomainCreateCustomDomain.UpdatedAt, and is useful for accessing the field via an interface.
func (v *CustomDomainCreateCustomDomainCreateCustomDomain) GetUpdatedAt() *time.Time {
	return v.UpdatedAt
}

// GetServiceId returns CustomDomainCreateCustomDomainCreateCustomDomain.ServiceId, and is useful for accessing the field via an interface.
func (v *CustomDomainCreateCustomDomainCreateCustomDomain) GetServiceId() string { return v.ServiceId }

// GetEnvironmentId returns CustomDomainCreateCustomDomainCreateCustomDomain.EnvironmentId, and is useful for accessing the field via an interface.
func (v *CustomDomainCreateCustomDomainCreateCustomDomain) GetEnvironmentId() string {
	return v.EnvironmentId
}

// GetStatus returns CustomDomainCreateCustomDomainCreateCustomDomain.Status, and is useful for accessing the field via an interface.
func (v *CustomDomainCreateCustomDomainCreateCustomDomain) GetStatus() *CustomDomainCreateCustomDomainCreateCustomDomainStatus {
	return v.Status
}

// CustomDomainCreateCustomDomainCreateCustomDomainStatus includes the requested fields of the GraphQL type CustomDomainStatus.
type CustomDomainCreateCustomDomainCreateCustomDomainStatus struct {
	DnsRecords        []*CustomDomainCreateCustomDomainCreateCustomDomainStatusDnsRecordsDNSRecords              `json:"dnsRecords"`
	CdnProvider       *CDNProvider                                                                               `json:"cdnProvider"`
	Certificates      []*CustomDomainCreateCustomDomainCreateCustomDomainStatusCertificatesCertificatePublicData `json:"certificates"`
	CertificateStatus CertificateStatus                                                                          `json:"certificateStatus"`
}

// GetDnsRecords returns CustomDomainCreateCustomDomainCreateCustomDomainStatus.DnsRecords, and is useful for accessing the field via an interface.
func (v *CustomDomainCreateCustomDomainCreateCustomDomainStatus) GetDnsRecords() []*CustomDomainCreateCustomDomainCreateCustomDomainStatusDnsRecordsDNSRecords {
	return v.DnsRecords
}

// GetCdnProvider returns CustomDomainCreateCustomDomainCreateCustomDomainStatus.CdnProvider, and is useful for accessing the field via an interface.
func (v *CustomDomainCreateCustomDomainCreateCustomDomainStatus) GetCdnProvider() *CDNProvider {
	return v.CdnProvider
}

// GetCertificates returns CustomDomainCreateCustomDomainCreateCustomDomainStatus.Certificates, and is useful for accessing the field via an interface.
func (v *CustomDomainCreateCustomDomainCreateCustomDomainStatus) GetCertificates() []*CustomDomainCreateCustomDomainCreateCustomDomainStatusCertificatesCertificatePublicData {
	return v.Certificates
}

// GetCertificateStatus returns CustomDomainCreateCustomDomainCreateCustomDomainStatus.CertificateStatus, and is useful for accessing the field via an interface.
func (v *CustomDomainCreateCustomDomainCreateCustomDomainStatus) GetCertificateStatus() CertificateStatus {
	return v.CertificateStatus
}

// CustomDomainCreateCustomDomainCreateCustomDomainStatusCertificatesCertificatePublicData includes the requested fields of the GraphQL type CertificatePublicData.
type CustomDomainCreateCustomDomainCreateCustomDomainStatusCertificatesCertificatePublicData struct {
	IssuedAt          *time.Time `json:"issuedAt"`
	ExpiresAt         *time.Time `json:"expiresAt"`
	DomainNames       []string   `json:"domainNames"`
	FingerprintSha256 string     `json:"fingerprintSha256"`
	KeyType           KeyType    `json:"keyType"`
}

// GetIssuedAt returns CustomDomainCreateCustomDomainCreateCustomDomainStatusCertificatesCertificatePublicData.IssuedAt, and is useful for accessing the field via an interface.
func (v *CustomDomainCreateCustomDomainCreateCustomDomainStatusCertificatesCertificatePublicData) GetIssuedAt() *time.Time {
	return v.IssuedAt
}

// GetExpiresAt returns CustomDomainCreateCustomDomainCreateCustomDomainStatusCertificatesCertificatePublicData.ExpiresAt, and is useful for accessing the field via an interface.
func (v *CustomDomainCreateCustomDomainCreateCustomDomainStatusCertificatesCertificatePublicData) GetExpiresAt() *time.Time {
	return v.ExpiresAt
}

// GetDomainNames returns CustomDomainCreateCustomDomainCreateCustomDomainStatusCertificatesCertificatePublicData.DomainNames, and is useful for accessing the field via an interface.
func (v *CustomDomainCreateCustomDomainCreateCustomDomainStatusCertificatesCertificatePublicData) GetDomainNames() []string {
	return v.DomainNames
}

// GetFingerprintSha256 returns CustomDomainCreateCustomDomainCreateCustomDomainStatusCertificatesCertificatePublicData.FingerprintSha256, and is useful for accessing the field via an interface.
func (v *CustomDomainCreateCustomDomainCreateCustomDomainStatusCertificatesCertificatePublicData) GetFingerprintSha256() string {
	return v.FingerprintSha256
}

// GetKeyType returns CustomDomainCreateCustomDomainCreateCustomDomainStatusCertificatesCertificatePublicData.KeyType, and is useful for accessing the field via an interface.
func (v *CustomDomainCreateCustomDomainCreateCustomDomainStatusCertificatesCertificatePublicData) GetKeyType() KeyType {
	return v.KeyType
}

// CustomDomainCreateCustomDomainCreateCustomDomainStatusDnsRecordsDNSRecords includes the requested fields of the GraphQL type DNSRecords.
type CustomDomainCreateCustomDomainCreateCustomDomainStatusDnsRecordsDNSRecords struct {
	Hostlabel     string           `json:"hostlabel"`
	Fqdn          string           `json:"fqdn"`
	RecordType    DNSRecordType    `json:"recordType"`
	RequiredValue string           `json:"requiredValue"`
	CurrentValue  string           `json:"currentValue"`
	Status        DNSRecordStatus  `json:"status"`
	Zone          string           `json:"zone"`
	Purpose       DNSRecordPurpose `json:"purpose"`
}

// GetHostlabel returns CustomDomainCreateCustomDomainCreateCustomDomainStatusDnsRecordsDNSRecords.Hostlabel, and is useful for accessing the field via an interface.
func (v *CustomDomainCreateCustomDomainCreateCustomDomainStatusDnsRecordsDNSRecords) GetHostlabel() string {
	return v.Hostlabel
}

// GetFqdn returns CustomDomainCreateCustomDomainCreateCustomDomainStatusDnsRecordsDNSRecords.Fqdn, and is useful for accessing the field via an interface.
func (v *CustomDomainCreateCustomDomainCreateCustomDomainStatusDnsRecordsDNSRecords) GetFqdn() string {
	return v.Fqdn
}

// GetRecordType returns CustomDomainCreateCustomDomainCreateCustomDomainStatusDnsRecordsDNSRecords.RecordType, and is useful for accessing the field via an interface.
func (v *CustomDomainCreateCustomDomainCreateCustomDomainStatusDnsRecordsDNSRecords) GetRecordType() DNSRecordType {
	return v.RecordType
}

// GetRequiredValue returns CustomDomainCreateCustomDomainCreateCustomDomainStatusDnsRecordsDNSRecords.RequiredValue, and is useful for accessing the field via an interface.
func (v *CustomDomainCreateCustomDomainCreateCustomDomainStatusDnsRecordsDNSRecords) GetRequiredValue() string {
	return v.RequiredValue
}

// GetCurrentValue returns CustomDomainCreateCustomDomainCreateCustomDomainStatusDnsRecordsDNSRecords.CurrentValue, and is useful for accessing the field via an interface.
func (v *CustomDomainCreateCustomDomainCreateCustomDomainStatusDnsRecordsDNSRecords) GetCurrentValue() string {
	return v.CurrentValue
}

// GetStatus returns CustomDomainCreateCustomDomainCreateCustomDomainStatusDnsRecordsDNSRecords.Status, and is useful for accessing the field via an interface.
func (v *CustomDomainCreateCustomDomainCreateCustomDomainStatusDnsRecordsDNSRecords) GetStatus() DNSRecordStatus {
	return v.Status
}

// GetZone returns CustomDomainCreateCustomDomainCreateCustomDomainStatusDnsRecordsDNSRecords.Zone, and is useful for accessing the field via an interface.
func (v *CustomDomainCreateCustomDomainCreateCustomDomainStatusDnsRecordsDNSRecords) GetZone() string {
	return v.Zone
}

// GetPurpose returns CustomDomainCreateCustomDomainCreateCustomDomainStatusDnsRecordsDNSRecords.Purpose, and is useful for accessing the field via an interface.
func (v *CustomDomainCreateCustomDomainCreateCustomDomainStatusDnsRecordsDNSRecords) GetPurpose() DNSRecordPurpose {
	return v.Purpose
}

type CustomDomainCreateInput struct {
	Domain        string `json:"domain"`
	EnvironmentId string `json:"environmentId"`
	ServiceId     string `json:"serviceId"`
}

// GetDomain returns CustomDomainCreateInput.Domain, and is useful for accessing the field via an interface.
func (v *CustomDomainCreateInput) GetDomain() string { return v.Domain }

// GetEnvironmentId returns CustomDomainCreateInput.EnvironmentId, and is useful for accessing the field via an interface.
func (v *CustomDomainCreateInput) GetEnvironmentId() string { return v.EnvironmentId }

// GetServiceId returns CustomDomainCreateInput.ServiceId, and is useful for accessing the field via an interface.
func (v *CustomDomainCreateInput) GetServiceId() string { return v.ServiceId }

// CustomDomainCreateResponse is returned by CustomDomainCreate on success.
type CustomDomainCreateResponse struct {
	// Creates a new custom domain.
	CustomDomainCreate *CustomDomainCreateCustomDomainCreateCustomDomain `json:"customDomainCreate"`
}

// GetCustomDomainCreate returns CustomDomainCreateResponse.CustomDomainCreate, and is useful for accessing the field via an interface.
func (v *CustomDomainCreateResponse) GetCustomDomainCreate() *CustomDomainCreateCustomDomainCreateCustomDomain {
	return v.CustomDomainCreate
}

type DNSRecordPurpose string

const (
	DNSRecordPurposeDnsRecordPurposeAcmeDns01Challenge DNSRecordPurpose = "DNS_RECORD_PURPOSE_ACME_DNS01_CHALLENGE"
	DNSRecordPurposeDnsRecordPurposeTrafficRoute       DNSRecordPurpose = "DNS_RECORD_PURPOSE_TRAFFIC_ROUTE"
	DNSRecordPurposeDnsRecordPurposeUnspecified        DNSRecordPurpose = "DNS_RECORD_PURPOSE_UNSPECIFIED"
	DNSRecordPurposeUnrecognized                       DNSRecordPurpose = "UNRECOGNIZED"
)

type DNSRecordStatus string

const (
	DNSRecordStatusDnsRecordStatusPropagated     DNSRecordStatus = "DNS_RECORD_STATUS_PROPAGATED"
	DNSRecordStatusDnsRecordStatusRequiresUpdate DNSRecordStatus = "DNS_RECORD_STATUS_REQUIRES_UPDATE"
	DNSRecordStatusDnsRecordStatusUnspecified    DNSRecordStatus = "DNS_RECORD_STATUS_UNSPECIFIED"
	DNSRecordStatusUnrecognized                  DNSRecordStatus = "UNRECOGNIZED"
)

type DNSRecordType string

const (
	DNSRecordTypeDnsRecordTypeA           DNSRecordType = "DNS_RECORD_TYPE_A"
	DNSRecordTypeDnsRecordTypeCname       DNSRecordType = "DNS_RECORD_TYPE_CNAME"
	DNSRecordTypeDnsRecordTypeNs          DNSRecordType = "DNS_RECORD_TYPE_NS"
	DNSRecordTypeDnsRecordTypeUnspecified DNSRecordType = "DNS_RECORD_TYPE_UNSPECIFIED"
	DNSRecordTypeUnrecognized             DNSRecordType = "UNRECOGNIZED"
)

type KeyType string

const (
	KeyTypeKeyTypeEcdsa       KeyType = "KEY_TYPE_ECDSA"
	KeyTypeKeyTypeRsa2048     KeyType = "KEY_TYPE_RSA_2048"
	KeyTypeKeyTypeRsa4096     KeyType = "KEY_TYPE_RSA_4096"
	KeyTypeKeyTypeUnspecified KeyType = "KEY_TYPE_UNSPECIFIED"
	KeyTypeUnrecognized       KeyType = "UNRECOGNIZED"
)

type RestartPolicyType string

const (
	RestartPolicyTypeAlways    RestartPolicyType = "ALWAYS"
	RestartPolicyTypeNever     RestartPolicyType = "NEVER"
	RestartPolicyTypeOnFailure RestartPolicyType = "ON_FAILURE"
)

type ServiceConnectInput struct {
	// The branch to connect to. e.g. 'main'
	Branch *string `json:"branch"`
	// The full name of the repo to connect to. e.g. 'railwayapp/starters'
	Repo *string `json:"repo"`
}

// GetBranch returns ServiceConnectInput.Branch, and is useful for accessing the field via an interface.
func (v *ServiceConnectInput) GetBranch() *string { return v.Branch }

// GetRepo returns ServiceConnectInput.Repo, and is useful for accessing the field via an interface.
func (v *ServiceConnectInput) GetRepo() *string { return v.Repo }

// ServiceConnectResponse is returned by ServiceConnect on success.
type ServiceConnectResponse struct {
	// Connect a service to a source
	ServiceConnect *ServiceConnectServiceConnectService `json:"serviceConnect"`
}

// GetServiceConnect returns ServiceConnectResponse.ServiceConnect, and is useful for accessing the field via an interface.
func (v *ServiceConnectResponse) GetServiceConnect() *ServiceConnectServiceConnectService {
	return v.ServiceConnect
}

// ServiceConnectServiceConnectService includes the requested fields of the GraphQL type Service.
type ServiceConnectServiceConnectService struct {
	Id        string    `json:"id"`
	Name      string    `json:"name"`
	Icon      *string   `json:"icon"`
	CreatedAt time.Time `json:"createdAt"`
	ProjectId string    `json:"projectId"`
}

// GetId returns ServiceConnectServiceConnectService.Id, and is useful for accessing the field via an interface.
func (v *ServiceConnectServiceConnectService) GetId() string { return v.Id }

// GetName returns ServiceConnectServiceConnectService.Name, and is useful for accessing the field via an interface.
func (v *ServiceConnectServiceConnectService) GetName() string { return v.Name }

// GetIcon returns ServiceConnectServiceConnectService.Icon, and is useful for accessing the field via an interface.
func (v *ServiceConnectServiceConnectService) GetIcon() *string { return v.Icon }

// GetCreatedAt returns ServiceConnectServiceConnectService.CreatedAt, and is useful for accessing the field via an interface.
func (v *ServiceConnectServiceConnectService) GetCreatedAt() time.Time { return v.CreatedAt }

// GetProjectId returns ServiceConnectServiceConnectService.ProjectId, and is useful for accessing the field via an interface.
func (v *ServiceConnectServiceConnectService) GetProjectId() string { return v.ProjectId }

type ServiceCreateInput struct {
	Branch *string `json:"branch"`
	// [Experimental] Environment ID. If the specified environment is a fork, the
	// service will only be created in it. Otherwise it will created in all
	// environments that are not forks of other environments
	EnvironmentId *string             `json:"environmentId"`
	Name          *string             `json:"name"`
	ProjectId     string              `json:"projectId"`
	Source        *ServiceSourceInput `json:"source,omitempty"`
	Variables     *map[string]string  `json:"variables"`
}

// GetBranch returns ServiceCreateInput.Branch, and is useful for accessing the field via an interface.
func (v *ServiceCreateInput) GetBranch() *string { return v.Branch }

// GetEnvironmentId returns ServiceCreateInput.EnvironmentId, and is useful for accessing the field via an interface.
func (v *ServiceCreateInput) GetEnvironmentId() *string { return v.EnvironmentId }

// GetName returns ServiceCreateInput.Name, and is useful for accessing the field via an interface.
func (v *ServiceCreateInput) GetName() *string { return v.Name }

// GetProjectId returns ServiceCreateInput.ProjectId, and is useful for accessing the field via an interface.
func (v *ServiceCreateInput) GetProjectId() string { return v.ProjectId }

// GetSource returns ServiceCreateInput.Source, and is useful for accessing the field via an interface.
func (v *ServiceCreateInput) GetSource() *ServiceSourceInput { return v.Source }

// GetVariables returns ServiceCreateInput.Variables, and is useful for accessing the field via an interface.
func (v *ServiceCreateInput) GetVariables() *map[string]string { return v.Variables }

// ServiceCreateResponse is returned by ServiceCreate on success.
type ServiceCreateResponse struct {
	// Creates a new service.
	ServiceCreate *ServiceCreateServiceCreateService `json:"serviceCreate"`
}

// GetServiceCreate returns ServiceCreateResponse.ServiceCreate, and is useful for accessing the field via an interface.
func (v *ServiceCreateResponse) GetServiceCreate() *ServiceCreateServiceCreateService {
	return v.ServiceCreate
}

// ServiceCreateServiceCreateService includes the requested fields of the GraphQL type Service.
type ServiceCreateServiceCreateService struct {
	Id        string    `json:"id"`
	Name      string    `json:"name"`
	Icon      *string   `json:"icon"`
	CreatedAt time.Time `json:"createdAt"`
	ProjectId string    `json:"projectId"`
}

// GetId returns ServiceCreateServiceCreateService.Id, and is useful for accessing the field via an interface.
func (v *ServiceCreateServiceCreateService) GetId() string { return v.Id }

// GetName returns ServiceCreateServiceCreateService.Name, and is useful for accessing the field via an interface.
func (v *ServiceCreateServiceCreateService) GetName() string { return v.Name }

// GetIcon returns ServiceCreateServiceCreateService.Icon, and is useful for accessing the field via an interface.
func (v *ServiceCreateServiceCreateService) GetIcon() *string { return v.Icon }

// GetCreatedAt returns ServiceCreateServiceCreateService.CreatedAt, and is useful for accessing the field via an interface.
func (v *ServiceCreateServiceCreateService) GetCreatedAt() time.Time { return v.CreatedAt }

// GetProjectId returns ServiceCreateServiceCreateService.ProjectId, and is useful for accessing the field via an interface.
func (v *ServiceCreateServiceCreateService) GetProjectId() string { return v.ProjectId }

// ServiceDeleteResponse is returned by ServiceDelete on success.
type ServiceDeleteResponse struct {
	// Deletes a service.
	ServiceDelete bool `json:"serviceDelete"`
}

// GetServiceDelete returns ServiceDeleteResponse.ServiceDelete, and is useful for accessing the field via an interface.
func (v *ServiceDeleteResponse) GetServiceDelete() bool { return v.ServiceDelete }

type ServiceInstanceUpdateInput struct {
	BuildCommand            *string             `json:"buildCommand"`
	Builder                 *Builder            `json:"builder"`
	CronSchedule            *string             `json:"cronSchedule"`
	HealthcheckPath         *string             `json:"healthcheckPath"`
	HealthcheckTimeout      *int                `json:"healthcheckTimeout"`
	NixpacksPlan            *map[string]string  `json:"nixpacksPlan"`
	RailwayConfigFile       *string             `json:"railwayConfigFile"`
	RestartPolicyMaxRetries *int                `json:"restartPolicyMaxRetries"`
	RestartPolicyType       *RestartPolicyType  `json:"restartPolicyType"`
	RootDirectory           *string             `json:"rootDirectory"`
	Source                  *ServiceSourceInput `json:"source,omitempty"`
	StartCommand            *string             `json:"startCommand"`
	WatchPatterns           []string            `json:"watchPatterns"`
}

// GetBuildCommand returns ServiceInstanceUpdateInput.BuildCommand, and is useful for accessing the field via an interface.
func (v *ServiceInstanceUpdateInput) GetBuildCommand() *string { return v.BuildCommand }

// GetBuilder returns ServiceInstanceUpdateInput.Builder, and is useful for accessing the field via an interface.
func (v *ServiceInstanceUpdateInput) GetBuilder() *Builder { return v.Builder }

// GetCronSchedule returns ServiceInstanceUpdateInput.CronSchedule, and is useful for accessing the field via an interface.
func (v *ServiceInstanceUpdateInput) GetCronSchedule() *string { return v.CronSchedule }

// GetHealthcheckPath returns ServiceInstanceUpdateInput.HealthcheckPath, and is useful for accessing the field via an interface.
func (v *ServiceInstanceUpdateInput) GetHealthcheckPath() *string { return v.HealthcheckPath }

// GetHealthcheckTimeout returns ServiceInstanceUpdateInput.HealthcheckTimeout, and is useful for accessing the field via an interface.
func (v *ServiceInstanceUpdateInput) GetHealthcheckTimeout() *int { return v.HealthcheckTimeout }

// GetNixpacksPlan returns ServiceInstanceUpdateInput.NixpacksPlan, and is useful for accessing the field via an interface.
func (v *ServiceInstanceUpdateInput) GetNixpacksPlan() *map[string]string { return v.NixpacksPlan }

// GetRailwayConfigFile returns ServiceInstanceUpdateInput.RailwayConfigFile, and is useful for accessing the field via an interface.
func (v *ServiceInstanceUpdateInput) GetRailwayConfigFile() *string { return v.RailwayConfigFile }

// GetRestartPolicyMaxRetries returns ServiceInstanceUpdateInput.RestartPolicyMaxRetries, and is useful for accessing the field via an interface.
func (v *ServiceInstanceUpdateInput) GetRestartPolicyMaxRetries() *int {
	return v.RestartPolicyMaxRetries
}

// GetRestartPolicyType returns ServiceInstanceUpdateInput.RestartPolicyType, and is useful for accessing the field via an interface.
func (v *ServiceInstanceUpdateInput) GetRestartPolicyType() *RestartPolicyType {
	return v.RestartPolicyType
}

// GetRootDirectory returns ServiceInstanceUpdateInput.RootDirectory, and is useful for accessing the field via an interface.
func (v *ServiceInstanceUpdateInput) GetRootDirectory() *string { return v.RootDirectory }

// GetSource returns ServiceInstanceUpdateInput.Source, and is useful for accessing the field via an interface.
func (v *ServiceInstanceUpdateInput) GetSource() *ServiceSourceInput { return v.Source }

// GetStartCommand returns ServiceInstanceUpdateInput.StartCommand, and is useful for accessing the field via an interface.
func (v *ServiceInstanceUpdateInput) GetStartCommand() *string { return v.StartCommand }

// GetWatchPatterns returns ServiceInstanceUpdateInput.WatchPatterns, and is useful for accessing the field via an interface.
func (v *ServiceInstanceUpdateInput) GetWatchPatterns() []string { return v.WatchPatterns }

// ServiceInstanceUpdateResponse is returned by ServiceInstanceUpdate on success.
type ServiceInstanceUpdateResponse struct {
	// Update a service instance
	ServiceInstanceUpdate bool `json:"serviceInstanceUpdate"`
}

// GetServiceInstanceUpdate returns ServiceInstanceUpdateResponse.ServiceInstanceUpdate, and is useful for accessing the field via an interface.
func (v *ServiceInstanceUpdateResponse) GetServiceInstanceUpdate() bool {
	return v.ServiceInstanceUpdate
}

type ServiceSourceInput struct {
	Image *string `json:"image"`
	Repo  *string `json:"repo"`
}

// GetImage returns ServiceSourceInput.Image, and is useful for accessing the field via an interface.
func (v *ServiceSourceInput) GetImage() *string { return v.Image }

// GetRepo returns ServiceSourceInput.Repo, and is useful for accessing the field via an interface.
func (v *ServiceSourceInput) GetRepo() *string { return v.Repo }

type VariableCollectionUpsertInput struct {
	EnvironmentId string `json:"environmentId"`
	ProjectId     string `json:"projectId"`
	// When set to true, removes all existing variables before upserting the new collection.
	Replace   *bool             `json:"replace"`
	ServiceId *string           `json:"serviceId"`
	Variables map[string]string `json:"variables"`
}

// GetEnvironmentId returns VariableCollectionUpsertInput.EnvironmentId, and is useful for accessing the field via an interface.
func (v *VariableCollectionUpsertInput) GetEnvironmentId() string { return v.EnvironmentId }

// GetProjectId returns VariableCollectionUpsertInput.ProjectId, and is useful for accessing the field via an interface.
func (v *VariableCollectionUpsertInput) GetProjectId() string { return v.ProjectId }

// GetReplace returns VariableCollectionUpsertInput.Replace, and is useful for accessing the field via an interface.
func (v *VariableCollectionUpsertInput) GetReplace() *bool { return v.Replace }

// GetServiceId returns VariableCollectionUpsertInput.ServiceId, and is useful for accessing the field via an interface.
func (v *VariableCollectionUpsertInput) GetServiceId() *string { return v.ServiceId }

// GetVariables returns VariableCollectionUpsertInput.Variables, and is useful for accessing the field via an interface.
func (v *VariableCollectionUpsertInput) GetVariables() map[string]string { return v.Variables }

// VariableCollectionUpsertResponse is returned by VariableCollectionUpsert on success.
type VariableCollectionUpsertResponse struct {
	// Upserts a collection of variables.
	VariableCollectionUpsert bool `json:"variableCollectionUpsert"`
}

// GetVariableCollectionUpsert returns VariableCollectionUpsertResponse.VariableCollectionUpsert, and is useful for accessing the field via an interface.
func (v *VariableCollectionUpsertResponse) GetVariableCollectionUpsert() bool {
	return v.VariableCollectionUpsert
}

// __CustomDomainCreateInput is used internally by genqlient
type __CustomDomainCreateInput struct {
	Input *CustomDomainCreateInput `json:"input,omitempty"`
}

// GetInput returns __CustomDomainCreateInput.Input, and is useful for accessing the field via an interface.
func (v *__CustomDomainCreateInput) GetInput() *CustomDomainCreateInput { return v.Input }

// __ServiceConnectInput is used internally by genqlient
type __ServiceConnectInput struct {
	Id    string               `json:"id"`
	Input *ServiceConnectInput `json:"input,omitempty"`
}

// GetId returns __ServiceConnectInput.Id, and is useful for accessing the field via an interface.
func (v *__ServiceConnectInput) GetId() string { return v.Id }

// GetInput returns __ServiceConnectInput.Input, and is useful for accessing the field via an interface.
func (v *__ServiceConnectInput) GetInput() *ServiceConnectInput { return v.Input }

// __ServiceCreateInput is used internally by genqlient
type __ServiceCreateInput struct {
	Input *ServiceCreateInput `json:"input,omitempty"`
}

// GetInput returns __ServiceCreateInput.Input, and is useful for accessing the field via an interface.
func (v *__ServiceCreateInput) GetInput() *ServiceCreateInput { return v.Input }

// __ServiceDeleteInput is used internally by genqlient
type __ServiceDeleteInput struct {
	Id            string  `json:"id"`
	EnvironmentId *string `json:"environmentId"`
}

// GetId returns __ServiceDeleteInput.Id, and is useful for accessing the field via an interface.
func (v *__ServiceDeleteInput) GetId() string { return v.Id }

// GetEnvironmentId returns __ServiceDeleteInput.EnvironmentId, and is useful for accessing the field via an interface.
func (v *__ServiceDeleteInput) GetEnvironmentId() *string { return v.EnvironmentId }

// __ServiceInstanceUpdateInput is used internally by genqlient
type __ServiceInstanceUpdateInput struct {
	ServiceId     string                      `json:"serviceId"`
	EnvironmentId *string                     `json:"environmentId"`
	Input         *ServiceInstanceUpdateInput `json:"input,omitempty"`
}

// GetServiceId returns __ServiceInstanceUpdateInput.ServiceId, and is useful for accessing the field via an interface.
func (v *__ServiceInstanceUpdateInput) GetServiceId() string { return v.ServiceId }

// GetEnvironmentId returns __ServiceInstanceUpdateInput.EnvironmentId, and is useful for accessing the field via an interface.
func (v *__ServiceInstanceUpdateInput) GetEnvironmentId() *string { return v.EnvironmentId }

// GetInput returns __ServiceInstanceUpdateInput.Input, and is useful for accessing the field via an interface.
func (v *__ServiceInstanceUpdateInput) GetInput() *ServiceInstanceUpdateInput { return v.Input }

// __VariableCollectionUpsertInput is used internally by genqlient
type __VariableCollectionUpsertInput struct {
	Input *VariableCollectionUpsertInput `json:"input,omitempty"`
}

// GetInput returns __VariableCollectionUpsertInput.Input, and is useful for accessing the field via an interface.
func (v *__VariableCollectionUpsertInput) GetInput() *VariableCollectionUpsertInput { return v.Input }

// The query or mutation executed by CustomDomainCreate.
const CustomDomainCreate_Operation = `
mutation CustomDomainCreate ($input: CustomDomainCreateInput!) {
	customDomainCreate(input: $input) {
		id
		domain
		createdAt
		updatedAt
		serviceId
		environmentId
		status {
			dnsRecords {
				hostlabel
				fqdn
				recordType
				requiredValue
				currentValue
				status
				zone
				purpose
			}
			cdnProvider
			certificates {
				issuedAt
				expiresAt
				domainNames
				fingerprintSha256
				keyType
			}
			certificateStatus
		}
	}
}
`

func CustomDomainCreate(
	client graphql.Client,
	input *CustomDomainCreateInput,
) (*CustomDomainCreateResponse, map[string]interface{}, error) {
	req := &graphql.Request{
		OpName: "CustomDomainCreate",
		Query:  CustomDomainCreate_Operation,
		Variables: &__CustomDomainCreateInput{
			Input: input,
		},
	}
	var err error

	var data CustomDomainCreateResponse
	resp := &graphql.Response{Data: &data}

	err = client.MakeRequest(
		nil,
		req,
		resp,
	)

	return &data, resp.Extensions, err
}

// The query or mutation executed by ServiceConnect.
const ServiceConnect_Operation = `
mutation ServiceConnect ($id: String!, $input: ServiceConnectInput!) {
	serviceConnect(id: $id, input: $input) {
		id
		name
		icon
		createdAt
		projectId
	}
}
`

func ServiceConnect(
	client graphql.Client,
	id string,
	input *ServiceConnectInput,
) (*ServiceConnectResponse, map[string]interface{}, error) {
	req := &graphql.Request{
		OpName: "ServiceConnect",
		Query:  ServiceConnect_Operation,
		Variables: &__ServiceConnectInput{
			Id:    id,
			Input: input,
		},
	}
	var err error

	var data ServiceConnectResponse
	resp := &graphql.Response{Data: &data}

	err = client.MakeRequest(
		nil,
		req,
		resp,
	)

	return &data, resp.Extensions, err
}

// The query or mutation executed by ServiceCreate.
const ServiceCreate_Operation = `
mutation ServiceCreate ($input: ServiceCreateInput!) {
	serviceCreate(input: $input) {
		id
		name
		icon
		createdAt
		projectId
	}
}
`

func ServiceCreate(
	client graphql.Client,
	input *ServiceCreateInput,
) (*ServiceCreateResponse, map[string]interface{}, error) {
	req := &graphql.Request{
		OpName: "ServiceCreate",
		Query:  ServiceCreate_Operation,
		Variables: &__ServiceCreateInput{
			Input: input,
		},
	}
	var err error

	var data ServiceCreateResponse
	resp := &graphql.Response{Data: &data}

	err = client.MakeRequest(
		nil,
		req,
		resp,
	)

	return &data, resp.Extensions, err
}

// The query or mutation executed by ServiceDelete.
const ServiceDelete_Operation = `
mutation ServiceDelete ($id: String!, $environmentId: String) {
	serviceDelete(id: $id, environmentId: $environmentId)
}
`

func ServiceDelete(
	client graphql.Client,
	id string,
	environmentId *string,
) (*ServiceDeleteResponse, map[string]interface{}, error) {
	req := &graphql.Request{
		OpName: "ServiceDelete",
		Query:  ServiceDelete_Operation,
		Variables: &__ServiceDeleteInput{
			Id:            id,
			EnvironmentId: environmentId,
		},
	}
	var err error

	var data ServiceDeleteResponse
	resp := &graphql.Response{Data: &data}

	err = client.MakeRequest(
		nil,
		req,
		resp,
	)

	return &data, resp.Extensions, err
}

// The query or mutation executed by ServiceInstanceUpdate.
const ServiceInstanceUpdate_Operation = `
mutation ServiceInstanceUpdate ($serviceId: String!, $environmentId: String, $input: ServiceInstanceUpdateInput!) {
	serviceInstanceUpdate(serviceId: $serviceId, environmentId: $environmentId, input: $input)
}
`

func ServiceInstanceUpdate(
	client graphql.Client,
	serviceId string,
	environmentId *string,
	input *ServiceInstanceUpdateInput,
) (*ServiceInstanceUpdateResponse, map[string]interface{}, error) {
	req := &graphql.Request{
		OpName: "ServiceInstanceUpdate",
		Query:  ServiceInstanceUpdate_Operation,
		Variables: &__ServiceInstanceUpdateInput{
			ServiceId:     serviceId,
			EnvironmentId: environmentId,
			Input:         input,
		},
	}
	var err error

	var data ServiceInstanceUpdateResponse
	resp := &graphql.Response{Data: &data}

	err = client.MakeRequest(
		nil,
		req,
		resp,
	)

	return &data, resp.Extensions, err
}

// The query or mutation executed by VariableCollectionUpsert.
const VariableCollectionUpsert_Operation = `
mutation VariableCollectionUpsert ($input: VariableCollectionUpsertInput!) {
	variableCollectionUpsert(input: $input)
}
`

func VariableCollectionUpsert(
	client graphql.Client,
	input *VariableCollectionUpsertInput,
) (*VariableCollectionUpsertResponse, map[string]interface{}, error) {
	req := &graphql.Request{
		OpName: "VariableCollectionUpsert",
		Query:  VariableCollectionUpsert_Operation,
		Variables: &__VariableCollectionUpsertInput{
			Input: input,
		},
	}
	var err error

	var data VariableCollectionUpsertResponse
	resp := &graphql.Response{Data: &data}

	err = client.MakeRequest(
		nil,
		req,
		resp,
	)

	return &data, resp.Extensions, err
}
