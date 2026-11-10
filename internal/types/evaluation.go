package types

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/yanyiwu/gojieba"
)

var jiebaDictionaryFiles = []string{
	"jieba.dict.utf8",
	"hmm_model.utf8",
	"user.dict.utf8",
	"idf.utf8",
	"stop_words.utf8",
}

// Jieba is a global instance of Chinese text segmentation tool.
// Dictionary files are resolved in this order:
//  1. JIEBA_DICT_DIR (explicit override)
//  2. resources/jieba next to the executable (Lite release layout)
//  3. resources/jieba under the current working directory (development)
//  4. gojieba's bundled module dictionary directory (development fallback)
var Jieba *gojieba.Jieba = newJieba()

func newJieba() *gojieba.Jieba {
	candidates := make([]string, 0, 4)

	if configuredDir := strings.TrimSpace(os.Getenv("JIEBA_DICT_DIR")); configuredDir != "" {
		candidates = append(candidates, configuredDir)
	}

	if executable, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Join(filepath.Dir(executable), "resources", "jieba"))
	}

	if workingDir, err := os.Getwd(); err == nil {
		candidates = append(candidates, filepath.Join(workingDir, "resources", "jieba"))
	}

	// gojieba derives this path from its source/module location. It is useful
	// during local development, but must not be the only path in a release.
	if moduleDictDir := strings.TrimSpace(gojieba.DICT_DIR); moduleDictDir != "" {
		candidates = append(candidates, moduleDictDir)
	}

	seen := make(map[string]struct{}, len(candidates))
	searched := make([]string, 0, len(candidates))
	for _, dir := range candidates {
		dir = filepath.Clean(dir)
		if _, ok := seen[dir]; ok {
			continue
		}
		seen[dir] = struct{}{}
		searched = append(searched, dir)

		if !hasJiebaDictionary(dir) {
			continue
		}

		paths := make([]string, 0, len(jiebaDictionaryFiles))
		for _, name := range jiebaDictionaryFiles {
			paths = append(paths, filepath.Join(dir, name))
		}
		return gojieba.NewJieba(paths...)
	}

	panic(fmt.Sprintf(
		"Jieba dictionary files not found. Put the five dictionary files in resources/jieba next to the executable, or set JIEBA_DICT_DIR. Searched: %s",
		strings.Join(searched, "; "),
	))
}

func hasJiebaDictionary(dir string) bool {
	if dir == "" {
		return false
	}
	for _, name := range jiebaDictionaryFiles {
		info, err := os.Stat(filepath.Join(dir, name))
		if err != nil || !info.Mode().IsRegular() {
			return false
		}
	}
	return true
}

// EvaluationStatue represents the status of an evaluation task.
type EvaluationStatue int

const (
	EvaluationStatuePending EvaluationStatue = iota // Task is waiting to start
	EvaluationStatueRunning                         // Task is in progress
	EvaluationStatueSuccess                         // Task completed successfully
	EvaluationStatueFailed                          // Task failed
)

// EvaluationTask contains information about an evaluation task.
type EvaluationTask struct {
	ID        string           `json:"id"`                // Unique task ID
	TenantID  uint64           `json:"tenant_id"`         // Tenant/Organization ID
	DatasetID string           `json:"dataset_id"`        // Dataset ID for evaluation
	StartTime time.Time        `json:"start_time"`        // Task start time
	Status    EvaluationStatue `json:"status"`            // Current task status
	ErrMsg    string           `json:"err_msg,omitempty"` // Error message if failed

	Total    int `json:"total,omitempty"`    // Total items to evaluate
	Finished int `json:"finished,omitempty"` // Finished items count
}

// EvaluationDetail contains detailed information about an evaluation task.
type EvaluationDetail struct {
	Task   *EvaluationTask `json:"task"`
	Params *ChatManage     `json:"params"`
	Metric *MetricResult   `json:"metric,omitempty"`
}

// String returns the JSON representation of EvaluationTask.
func (e *EvaluationTask) String() string {
	b, _ := json.Marshal(e)
	return string(b)
}

// MetricInput contains input data for metric calculation.
type MetricInput struct {
	RetrievalGT  [][]int
	RetrievalIDs []int

	GeneratedTexts string
	GeneratedGT    string
}

// MetricResult contains detailed metric results.
type MetricResult struct {
	RetrievalMetrics  RetrievalMetrics  `json:"retrieval_metrics"`
	GenerationMetrics GenerationMetrics `json:"generation_metrics"`
}

// RetrievalMetrics contains retrieval quality metrics.
type RetrievalMetrics struct {
	Precision float64 `json:"precision"`
	Recall    float64 `json:"recall"`
	NDCG3     float64 `json:"ndcg3"`
	NDCG10    float64 `json:"ndcg10"`
	MRR       float64 `json:"mrr"`
	MAP       float64 `json:"map"`
}

// GenerationMetrics contains generation quality metrics.
type GenerationMetrics struct {
	BLEU1  float64 `json:"bleu1"`
	BLEU2  float64 `json:"bleu2"`
	BLEU4  float64 `json:"bleu4"`
	ROUGE1 float64 `json:"rouge1"`
	ROUGE2 float64 `json:"rouge2"`
	ROUGEL float64 `json:"rougel"`
}

// EvalState represents different stages of an evaluation process.
type EvalState int

const (
	StateBegin EvalState = iota
	StateAfterQaPairs
	StateAfterDataset
	StateAfterEmbedding
	StateAfterVectorSearch
	StateAfterRerank
	StateAfterComplete
	StateEnd
)
