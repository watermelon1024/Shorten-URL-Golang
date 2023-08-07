package main

import (
	"embed"
	"encoding/json"
	"io/fs"
	"strings"
)

type route map[string]*route

// get routes from embed files
// check path is match `/\[[^/]*\]` ( for next.js export path format )
func getRoutes(dir embed.FS) route {
	routes, err := dir.ReadFile("dist/routes.json")
	if err != nil {
		return getDefaultRoutes(dir)
	}

	reading := route{}
	if err := json.Unmarshal(routes, &reading); err != nil {
		return getDefaultRoutes(dir)
	}

	return reading
}

func getDefaultRoutes(dir fs.FS) route {
	dPaths := route{}
	fs.WalkDir(dir, ".", func(path string, file fs.DirEntry, _ error) (err error) {
		if file.IsDir() {
			if strings.HasPrefix(path, "/") { // Make sure the regex test is correct
				path = "/" + path
			}
			if checkDynamicRoute.MatchString(path) {
				var t *route
				for i, p := range strings.Split(path, "/") {
					if i == 0 {
						route := &route{}
						dPaths[p] = route
						t = route
					} else {
						r := &route{}
						(*t)[p] = r
						t = r
					}
				}
			}
		}

		return // return nil
	})

	return dPaths
}

// check path is match route
func (s route) HasIs(path string) (bool, string) {
	var t route

	paths, resultPath := strings.Split(strings.TrimSuffix(strings.TrimPrefix(path, "/"), "/"), "/"), ""
	for i, p := range paths {
		if i == 0 {
			if _, ok := s[p]; ok {
				t = *s[p]             // default route map
				resultPath += "/" + p // add current path
				continue
			} else {
				return false, ""
			}
		}

		if _, ok := t[p]; ok {
			resultPath += "/" + p // add current path
			t = *t[p]

			if i == len(paths)-1 {
				return true, resultPath
			}
			continue
		}

		check := false
		// check dynamic route
		for key := range t {
			if strings.HasPrefix(key, "[") && strings.HasSuffix(key, "]") {
				t = *t[key]             // next route map
				resultPath += "/" + key // add current path
				if i == len(paths)-1 {
					return true, resultPath
				}
				check = true
				break
			}
		}
		if !check {
			return false, ""
		}
	}

	if resultPath != "" {
		return true, resultPath
	} else {
		return false, ""
	}
}
