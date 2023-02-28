package issue

import (
	"sync"
	"time"

	"github.com/stjudewashere/seonaut/internal/pagereport"
)

const (
	Error30x                         = iota + 1 // HTTP redirect
	Error40x                                    // HTTP not found
	Error50x                                    // HTTP internal error
	ErrorDuplicatedTitle                        // Duplicate title
	ErrorDuplicatedDescription                  // Duplicate description
	ErrorEmptyTitle                             // Missing or empty title
	ErrorShortTitle                             // Page title is too short
	ErrorLongTitle                              // Page title is too long
	ErrorEmptyDescription                       // Missing or empty meta description
	ErrorShortDescription                       // Meta description is too short
	ErrorLongDescription                        // Meta description is too long
	ErrorLittleContent                          // Not enough content
	ErrorImagesWithNoAlt                        // Images with no alt attribute
	ErrorRedirectChain                          // Redirect chain
	ErrorNoH1                                   // Missing or empy H1 tag
	ErrorNoLang                                 // Missing or empty html lang attribute
	ErrorHTTPLinks                              // Links using insecure http schema
	ErrorHreflangsReturnLink                    // Hreflang is not bidirectional
	ErrorTooManyLinks                           // Page contains too many links
	ErrorInternalNoFollow                       // Page has internal links with nofollow attribute
	ErrorExternalWithoutNoFollow                // Page has external follow links
	ErrorCanonicalizedToNonCanonical            // Page canonicalized to a non canonical page
	ErrorRedirectLoop                           // Redirect loop
	ErrorNotValidHeadings                       // H1-H6 tags have wrong order
	HreflangToNonCanonical                      // Hreflang to non canonical page
	ErrorInternalNoFollowIndexable              // Nofollow links to indexable pages
	ErrorNoIndexable                            // Page using the noindex attribute
	HreflangNoindexable                         // Hreflang to a non indexable page
	ErrorBlocked                                // Blocked by robots.txt
	ErrorOrphan                                 // Orphan pages
	SitemapNoIndex                              // No index pages included in the sitemap
	SitemapBlocked                              // Pages included in the sitemap that are blocked in robots.txt
	SitemapNonCanonical                         // Non canonical pages included in the sitemap
	IncomingFollowNofollow                      // Pages with index and noindex incoming links
)

type Reporter func(int64) <-chan *pagereport.PageReport

type IssueCallback struct {
	Callback  Reporter
	ErrorType int
}

type ReportManager struct {
	store     ReportManagerStore
	callbacks []IssueCallback
}

type ReportManagerStore interface {
	SaveIssues(<-chan *Issue)
	SaveEndIssues(int64, time.Time, int)
}

func NewReportManager(s ReportManagerStore) *ReportManager {
	return &ReportManager{
		store: s,
	}
}

// Add an issue reporter to the ReportManager.
// It will be used when creating the issues.
func (r *ReportManager) AddReporter(c Reporter, t int) {
	r.callbacks = append(r.callbacks, IssueCallback{Callback: c, ErrorType: t})
}

// CreateIssues uses the Reporters to create and save issues found in a crawl.
func (r *ReportManager) CreateIssues(cid int64) {
	issueCount := 0
	iStream := make(chan *Issue)

	wg := new(sync.WaitGroup)
	wg.Add(1)

	go func() {
		r.store.SaveIssues(iStream)
		wg.Done()
	}()

	for _, c := range r.callbacks {
		for p := range c.Callback(cid) {
			i := &Issue{
				PageReportId: p.Id,
				CrawlId:      cid,
				ErrorType:    c.ErrorType,
			}

			iStream <- i

			issueCount++
		}
	}

	close(iStream)

	wg.Wait()

	r.store.SaveEndIssues(cid, time.Now(), issueCount)
}
