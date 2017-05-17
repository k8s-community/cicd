package cicd

// Error is a common error typical for all responses
type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// BuildRequest defines request body of Build API method
type BuildRequest struct {
	Username   string `json:"username"`
	Repository string `json:"repository"`
	CommitHash string `json:"commitHash"`
}

// BuildResponse defines response body of Build API method
type BuildResponse struct {
	Error *Error `json:"error,omitempty"`
	Data  *Build `json:"data,omitempty"`
}

// Build defines data for response body of Build API method
// It contains RequestID parameter to be able to deal with logs of CICD service
type Build struct {
	RequestID string `json:"requestID"`
}
