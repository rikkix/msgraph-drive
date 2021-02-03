package drive

// LoginBaseURL represents the basic url used to acquire a token for the msgraph api
const LoginBaseURL string = "https://login.microsoftonline.com"

// BaseURL represents the URL used to perform all ms graph API-calls
const BaseURL string = "https://graph.microsoft.com"

// APIVersion represents the APIVersion of msgraph used by this implementation
const APIVersion string = "v1.0"

// MaxPageSize is the maximum Page size for an API-call. This will be rewritten to use paging some day. Currently limits environments to 999 entries (e.g. Users, CalendarEvents etc.)
const MaxPageSize int = 999
