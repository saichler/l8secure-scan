/*
© 2025 Sharon Aicler (saichler@gmail.com)

Layer 8 Ecosystem is licensed under the Apache License, Version 2.0.
You may obtain a copy of the License at:

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package dataimport

import (
	"github.com/saichler/l8types/go/types/l8api"
	"strings"
	"unicode"
)

// IAIMappingProvider defines the interface for AI-assisted column mapping.
type IAIMappingProvider interface {
	SuggestMapping(targetFields []*l8api.L8ImportFieldInfo, sourceColumns []string,
		sampleValues []string) (*l8api.L8ImportAIMappingResponse, error)
}

var aiProvider IAIMappingProvider = &HeuristicProvider{}

// SetAIMappingProvider sets a custom AI mapping provider (e.g., LLM-based).
func SetAIMappingProvider(p IAIMappingProvider) {
	if p != nil {
		aiProvider = p
	}
}

// GetAIMappingProvider returns the current AI mapping provider.
func GetAIMappingProvider() IAIMappingProvider {
	return aiProvider
}

// HeuristicProvider uses fuzzy string matching to suggest column mappings.
type HeuristicProvider struct{}

func (h *HeuristicProvider) SuggestMapping(targetFields []*l8api.L8ImportFieldInfo,
	sourceColumns []string, sampleValues []string) (*l8api.L8ImportAIMappingResponse, error) {

	var mappings []*l8api.L8ImportColumnMapping
	var totalConfidence float32
	matched := 0

	usedTargets := make(map[string]bool)

	for _, src := range sourceColumns {
		best := ""
		bestScore := float32(0)

		for _, tf := range targetFields {
			if tf.IsPrimaryKey || usedTargets[tf.FieldName] {
				continue
			}
			score := matchScore(src, tf.FieldName)
			if score > bestScore {
				bestScore = score
				best = tf.FieldName
			}
		}

		mapping := &l8api.L8ImportColumnMapping{SourceColumn: src}
		if bestScore >= 0.6 && best != "" {
			mapping.TargetField = best
			usedTargets[best] = true
			totalConfidence += bestScore
			matched++
		} else {
			mapping.Skip = true
		}
		mappings = append(mappings, mapping)
	}

	var confidence float32
	if matched > 0 {
		confidence = totalConfidence / float32(matched)
	}

	return &l8api.L8ImportAIMappingResponse{
		SuggestedMappings: mappings,
		Confidence:        confidence,
		Explanation:       "Mapped using heuristic string matching (name similarity, case normalization, abbreviation expansion).",
	}, nil
}

// matchScore returns a 0.0-1.0 confidence score for matching a source column to a target field.
func matchScore(source, target string) float32 {
	srcNorm := normalize(source)
	tgtNorm := normalize(target)

	// Exact match after normalization
	if srcNorm == tgtNorm {
		return 1.0
	}

	// Snake/camel normalization match
	srcTokens := tokenize(source)
	tgtTokens := tokenize(target)
	if strings.Join(srcTokens, "") == strings.Join(tgtTokens, "") {
		return 0.9
	}

	// Abbreviation expansion
	if matchesAbbreviation(srcTokens, tgtTokens) {
		return 0.8
	}

	// Substring containment
	if strings.Contains(srcNorm, tgtNorm) || strings.Contains(tgtNorm, srcNorm) {
		return 0.7
	}

	// Levenshtein distance
	dist := levenshtein(srcNorm, tgtNorm)
	maxLen := len(srcNorm)
	if len(tgtNorm) > maxLen {
		maxLen = len(tgtNorm)
	}
	if maxLen > 0 && dist <= 2 && float32(dist)/float32(maxLen) < 0.3 {
		return 0.6
	}

	return 0
}

// normalize converts to lowercase and removes non-alphanumeric characters.
func normalize(s string) string {
	var sb strings.Builder
	for _, r := range strings.ToLower(s) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			sb.WriteRune(r)
		}
	}
	return sb.String()
}

// tokenize splits a string into lowercase tokens by separators and camelCase boundaries.
func tokenize(s string) []string {
	// Replace common separators with space
	replaced := strings.NewReplacer("_", " ", "-", " ", ".", " ").Replace(s)

	// Split camelCase
	var tokens []string
	var current strings.Builder
	for i, r := range replaced {
		if r == ' ' {
			if current.Len() > 0 {
				tokens = append(tokens, strings.ToLower(current.String()))
				current.Reset()
			}
			continue
		}
		if unicode.IsUpper(r) && i > 0 && current.Len() > 0 {
			tokens = append(tokens, strings.ToLower(current.String()))
			current.Reset()
		}
		current.WriteRune(r)
	}
	if current.Len() > 0 {
		tokens = append(tokens, strings.ToLower(current.String()))
	}
	return tokens
}

// Common abbreviation mappings
var abbreviations = map[string]string{
	"emp":  "employee",
	"dept": "department",
	"mgr":  "manager",
	"addr": "address",
	"qty":  "quantity",
	"amt":  "amount",
	"desc": "description",
	"num":  "number",
	"no":   "number",
	"id":   "id",
	"fname": "first",
	"lname": "last",
	"tel":  "phone",
	"org":  "organization",
	"cat":  "category",
	"stat": "status",
	"dt":   "date",
}

// matchesAbbreviation checks if source tokens match target tokens after abbreviation expansion.
func matchesAbbreviation(srcTokens, tgtTokens []string) bool {
	expanded := make([]string, len(srcTokens))
	for i, t := range srcTokens {
		if exp, ok := abbreviations[t]; ok {
			expanded[i] = exp
		} else {
			expanded[i] = t
		}
	}
	return strings.Join(expanded, "") == strings.Join(tgtTokens, "")
}

// levenshtein computes the edit distance between two strings.
func levenshtein(a, b string) int {
	la, lb := len(a), len(b)
	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}
	prev := make([]int, lb+1)
	curr := make([]int, lb+1)
	for j := 0; j <= lb; j++ {
		prev[j] = j
	}
	for i := 1; i <= la; i++ {
		curr[0] = i
		for j := 1; j <= lb; j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			curr[j] = min(curr[j-1]+1, min(prev[j]+1, prev[j-1]+cost))
		}
		prev, curr = curr, prev
	}
	return prev[lb]
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
