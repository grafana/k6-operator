package cloud

// testRunList holds the output from /get-tests call
type testRunList struct {
	List []string `json:"list"`
}

// TestRunData holds the output from /get-test-data call
type TestRunData struct {
	TestRunId string `json:"testRunId"`
	// ArchiveURL
	Instances int `json:"instances"`
}
