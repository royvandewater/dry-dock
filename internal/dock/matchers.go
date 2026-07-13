package dock

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// lazyVersionQuery dumps each plugin's parsed version constraint as JSON. It
// runs inside lazy.vim so the constraints match exactly what lazy resolves,
// including any merged or nested spec fragments.
const lazyVersionQuery = `lua local o={}; for _,p in ipairs(require("lazy").plugins()) do if p.version then o[p.name]=p.version end end; io.write("\n"..vim.json.encode(o).."\n")`

// lazyMatchers asks lazy.vim, via a headless nvim, for the version constraint on
// each plugin. Plugins without a constraint are omitted. It is best-effort: a
// caller that can't reach nvim should fall back to treating every plugin as
// unconstrained.
func lazyMatchers() (map[string]string, error) {
	cmd := exec.Command("nvim", "--headless", "-c", lazyVersionQuery, "-c", "qa")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("querying lazy.vim for version matchers: %w\n%s", err, out)
	}
	return parseMatchers(string(out))
}

// parseMatchers extracts the JSON object the query prints. nvim mixes other
// startup output into the stream, so we pick the first line that parses as a
// JSON object.
func parseMatchers(out string) (map[string]string, error) {
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "{") {
			continue
		}
		matchers := map[string]string{}
		if err := json.Unmarshal([]byte(line), &matchers); err == nil {
			return matchers, nil
		}
	}
	return nil, fmt.Errorf("no version matcher JSON found in nvim output")
}
