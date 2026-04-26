package install

import "fmt"

// BrowserRevision holds version and download information for a browser build.
type BrowserRevision struct {
	Browser  string
	Revision string
	BaseURL  string
}

// knownRevisions maps browser names to their latest known-good revisions.
var knownRevisions = map[string]BrowserRevision{
	"chromium": {
		Browser:  "chromium",
		Revision: "1148",
		BaseURL:  "https://playwright.azureedge.net/builds/chromium",
	},
	"firefox": {
		Browser:  "firefox",
		Revision: "1467",
		BaseURL:  "https://playwright.azureedge.net/builds/firefox",
	},
	"webkit": {
		Browser:  "webkit",
		Revision: "2098",
		BaseURL:  "https://playwright.azureedge.net/builds/webkit",
	},
}

// LookupRevision returns the known-good revision for a browser.
func LookupRevision(browser string) (BrowserRevision, bool) {
	rev, ok := knownRevisions[browser]
	return rev, ok
}

// DownloadURL constructs the full download URL for a browser on a given platform.
func DownloadURL(browser string, platform Platform) (string, error) {
	rev, ok := LookupRevision(browser)
	if !ok {
		return "", fmt.Errorf("install: unknown browser %q", browser)
	}

	suffix := platform.Suffix()
	return fmt.Sprintf("%s/%s/%s-%s.zip", rev.BaseURL, rev.Revision, rev.Browser, suffix), nil
}

// SupportedBrowsers returns the list of browsers that can be installed.
func SupportedBrowsers() []string {
	return []string{"chromium", "firefox", "webkit"}
}
