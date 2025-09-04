package helpers

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// JUnitTestSuite represents the root element of a JUnit XML report
type JUnitTestSuite struct {
	XMLName   xml.Name        `xml:"testsuite"`
	Name      string          `xml:"name,attr"`
	Tests     int             `xml:"tests,attr"`
	Failures  int             `xml:"failures,attr"`
	Errors    int             `xml:"errors,attr"`
	Skipped   int             `xml:"skipped,attr"`
	Time      string          `xml:"time,attr"`
	Timestamp string          `xml:"timestamp,attr"`
	TestCases []JUnitTestCase `xml:"testcase"`
}

// JUnitTestCase represents a single test case in JUnit XML format
type JUnitTestCase struct {
	Name      string          `xml:"name,attr"`
	Classname string          `xml:"classname,attr"`
	Time      string          `xml:"time,attr"`
	Failure   *JUnitFailure   `xml:"failure,omitempty"`
	Error     *JUnitError     `xml:"error,omitempty"`
	Skipped   *JUnitSkipped   `xml:"skipped,omitempty"`
	SystemOut *JUnitSystemOut `xml:"system-out,omitempty"`
}

// JUnitFailure represents a test failure
type JUnitFailure struct {
	Message string `xml:"message,attr"`
	Type    string `xml:"type,attr"`
	Text    string `xml:",chardata"`
}

// JUnitError represents a test error
type JUnitError struct {
	Message string `xml:"message,attr"`
	Type    string `xml:"type,attr"`
	Text    string `xml:",chardata"`
}

// JUnitSkipped represents a skipped test
type JUnitSkipped struct {
	Message string `xml:"message,attr,omitempty"`
}

// JUnitSystemOut represents system output for a test
type JUnitSystemOut struct {
	Text string `xml:",chardata"`
}

// TestResult represents a test result that can be converted to JUnit XML
// This mirrors the structure from test_base_migration.go
type TestResult struct {
	Name     string
	Status   string
	Duration time.Duration
	Steps    []*TestStep
}

// TestStep represents a test step that can be converted to JUnit XML
type TestStep struct {
	Name     string
	Status   string
	Duration time.Duration
	Message  string
}

// GenerateJUnitXML creates a JUnit XML report from test results
func GenerateJUnitXML(testName string, vmName string, diskType string, startTime time.Time, results []*TestResult) (string, error) {
	// Calculate overall statistics
	totalFailures := 0
	totalErrors := 0
	totalSkipped := 0
	totalDuration := time.Since(startTime)

	// Create test cases for each test result and its steps
	var testCases []JUnitTestCase

	for _, result := range results {
		// Create a main test case for the overall test
		testCase := JUnitTestCase{
			Name:      result.Name,
			Classname: "vsphere.xcopy.e2e.tests",
			Time:      fmt.Sprintf("%.3f", result.Duration.Seconds()),
		}

		// Build system output with step details
		var systemOutBuilder strings.Builder
		systemOutBuilder.WriteString(fmt.Sprintf("Test: %s\n", result.Name))
		systemOutBuilder.WriteString(fmt.Sprintf("Status: %s\n", result.Status))
		systemOutBuilder.WriteString(fmt.Sprintf("Duration: %v\n", result.Duration))
		systemOutBuilder.WriteString(fmt.Sprintf("VM Name: %s\n", vmName))
		systemOutBuilder.WriteString(fmt.Sprintf("Disk Type: %s\n", diskType))
		systemOutBuilder.WriteString("\nSteps:\n")

		var failureMessages []string

		for _, step := range result.Steps {
			systemOutBuilder.WriteString(fmt.Sprintf("  - %s: %s (%v)\n", step.Name, step.Status, step.Duration))
			if step.Message != "" {
				systemOutBuilder.WriteString(fmt.Sprintf("    Message: %s\n", step.Message))
			}

			// Collect failure information
			if step.Status == "Failed" {
				failureMessages = append(failureMessages, fmt.Sprintf("%s: %s", step.Name, step.Message))
			}
		}

		testCase.SystemOut = &JUnitSystemOut{Text: systemOutBuilder.String()}

		// Set failure/error information based on test status
		switch result.Status {
		case "Failed":
			totalFailures++
			failureMessage := strings.Join(failureMessages, "; ")
			if failureMessage == "" {
				failureMessage = "Test failed"
			}
			testCase.Failure = &JUnitFailure{
				Message: failureMessage,
				Type:    "TestFailure",
				Text:    systemOutBuilder.String(),
			}
		case "Skipped":
			totalSkipped++
			testCase.Skipped = &JUnitSkipped{
				Message: "Test was skipped",
			}
		}

		testCases = append(testCases, testCase)

		// Create individual test cases for each step
		for _, step := range result.Steps {
			stepTestCase := JUnitTestCase{
				Name:      fmt.Sprintf("%s.%s", result.Name, step.Name),
				Classname: "vsphere.xcopy.e2e.steps",
				Time:      fmt.Sprintf("%.3f", step.Duration.Seconds()),
			}

			stepSystemOut := fmt.Sprintf("Step: %s\nStatus: %s\nDuration: %v\n", step.Name, step.Status, step.Duration)
			if step.Message != "" {
				stepSystemOut += fmt.Sprintf("Message: %s\n", step.Message)
			}
			stepTestCase.SystemOut = &JUnitSystemOut{Text: stepSystemOut}

			switch step.Status {
			case "Failed":
				stepTestCase.Failure = &JUnitFailure{
					Message: step.Message,
					Type:    "StepFailure",
					Text:    stepSystemOut,
				}
			case "Skipped":
				stepTestCase.Skipped = &JUnitSkipped{
					Message: step.Message,
				}
			}

			testCases = append(testCases, stepTestCase)
		}
	}

	// Create the test suite
	testSuite := JUnitTestSuite{
		Name:      fmt.Sprintf("E2E_Tests_%s", testName),
		Tests:     len(testCases),
		Failures:  totalFailures,
		Errors:    totalErrors,
		Skipped:   totalSkipped,
		Time:      fmt.Sprintf("%.3f", totalDuration.Seconds()),
		Timestamp: startTime.Format("2006-01-02T15:04:05"),
		TestCases: testCases,
	}

	// Generate XML
	xmlData, err := xml.MarshalIndent(testSuite, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal JUnit XML: %w", err)
	}

	// Add XML header
	xmlContent := xml.Header + string(xmlData)
	return xmlContent, nil
}

// SaveJUnitXMLToFile saves JUnit XML content to files in the logs directory
func SaveJUnitXMLToFile(xmlContent, testName string) error {
	logsDir := GetEnvOrDefault(EnvLogDir, DefaultLogDir)
	if err := os.MkdirAll(logsDir, DefaultDirPermissions); err != nil {
		return fmt.Errorf("failed to create logs directory: %w", err)
	}

	// Create JUnit XML file with timestamp
	timestamp := time.Now().Format("20060102_150405")
	junitFile := filepath.Join(logsDir, fmt.Sprintf("junit_%s_%s.xml", testName, timestamp))

	if err := writeToFile(junitFile, xmlContent); err != nil {
		return fmt.Errorf("failed to write JUnit XML file: %w", err)
	}

	// Also save to a standard location that Jenkins can easily find
	standardJunitFile := filepath.Join(logsDir, "junit-results.xml")
	if err := writeToFile(standardJunitFile, xmlContent); err != nil {
		return fmt.Errorf("failed to write JUnit XML to standard location: %w", err)
	}

	return nil
}

// writeToFile is a helper function to write content to a file
func writeToFile(filePath, content string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.WriteString(content)
	return err
}

// GetEnvOrDefault returns environment variable value or default if not set
func GetEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// GenerateJUnitReport is a convenience function that converts test framework results
// to JUnit XML format and saves them to files
func GenerateJUnitReport(testName, vmName, diskType string, startTime time.Time, results interface{}) error {
	// Convert results to the expected format
	var junitResults []*TestResult

	// Handle different types of results - this allows flexibility for different test framework structures
	switch v := results.(type) {
	case []*TestResult:
		junitResults = v
	default:
		return fmt.Errorf("unsupported results type: %T", results)
	}

	// Generate JUnit XML content
	xmlContent, err := GenerateJUnitXML(testName, vmName, diskType, startTime, junitResults)
	if err != nil {
		return fmt.Errorf("failed to generate JUnit XML: %w", err)
	}

	// Save JUnit XML to file
	if err := SaveJUnitXMLToFile(xmlContent, testName); err != nil {
		return fmt.Errorf("failed to save JUnit XML: %w", err)
	}

	return nil
}
