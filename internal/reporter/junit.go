package reporter

import (
	"encoding/xml"
	"fmt"
	"io"
)

type junitTestSuites struct {
	XMLName    xml.Name         `xml:"testsuites"`
	TestSuites []junitTestSuite `xml:"testsuite"`
}

type junitTestSuite struct {
	Name      string          `xml:"name,attr"`
	Tests     int             `xml:"tests,attr"`
	Failures  int             `xml:"failures,attr"`
	TestCases []junitTestCase `xml:"testcase"`
}

type junitTestCase struct {
	Name    string        `xml:"name,attr"`
	Failure *junitFailure `xml:"failure,omitempty"`
}

type junitFailure struct {
	Message string `xml:"message,attr"`
	Text    string `xml:",chardata"`
}

type JUnit struct {
	Writer io.Writer
}

func (j *JUnit) Render(r Report) error {
	w := effectiveWriter(j.Writer)

	suites := junitTestSuites{}
	for _, res := range r.Results {
		suite := junitTestSuite{
			Name:     fmt.Sprintf("Type %s", res.Type),
			Tests:    res.Success + res.Errors,
			Failures: res.Errors,
		}
		for i := 0; i < res.Success; i++ {
			suite.TestCases = append(suite.TestCases, junitTestCase{
				Name: fmt.Sprintf("valid_file_%d", i+1),
			})
		}
		for _, d := range res.Details {
			failureMsg := d.Path
			failureText := d.Message
			if d.Count > 1 {
				failureMsg = fmt.Sprintf("%d errors (run with --verbose for details)", d.Count)
			}
			suite.TestCases = append(suite.TestCases, junitTestCase{
				Name: d.File,
				Failure: &junitFailure{
					Message: failureMsg,
					Text:    failureText,
				},
			})
		}
		suites.TestSuites = append(suites.TestSuites, suite)
	}

	fmt.Fprint(w, xml.Header)
	enc := xml.NewEncoder(w)
	enc.Indent("", "  ")
	if err := enc.Encode(suites); err != nil {
		return err
	}
	return enc.Flush()
}
