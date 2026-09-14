package courseservice

import (
	"context"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/localcache"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/course"
)

type CatalogFacets struct {
	Departments []string
	Terms       []TermOption
	Campuses    []string
}

var catalogFacetsCache = &localcache.Cache[CatalogFacets]{MaxEntries: 1}

// Public filter dictionaries tolerate a short stale interval. They contain no
// user state, and request cancellation reaches each query on a cache miss.
func GetCatalogFacets(ctx context.Context) (CatalogFacets, error) {
	return catalogFacetsCache.GetOrLoadE("catalog", func() (CatalogFacets, error) {
		ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
		var value CatalogFacets
		var err error
		value.Departments, err = course.ListDistinctDepartmentsContext(ctx)
		if err != nil {
			return value, err
		}
		terms, err := course.ListDistinctTermsContext(ctx)
		if err != nil {
			return value, err
		}
		value.Terms = make([]TermOption, 0, len(terms))
		for _, term := range terms {
			label := term.Name
			if label == "" {
				label = term.Code
			}
			value.Terms = append(value.Terms, TermOption{Value: term.Code, Label: label})
		}
		value.Campuses, err = course.ListDistinctCampusesContext(ctx)
		return value, err
	}, 5*time.Minute)
}
