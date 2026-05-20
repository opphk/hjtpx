package captcha

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math/rand"
	"time"

	"github.com/hjtpx/hjtpx/internal/repository/cache"
	"github.com/hjtpx/hjtpx/internal/repository/db"
)

type SemanticGeneratorService struct {
	sessionCache *cache.SessionCache
	captchaRepo  *db.CaptchaRepository
	questionBank *SemanticQuestionBank
}

type CreateSemanticCaptchaRequest struct {
	Language    string `json:"language"`
	Difficulty  string `json:"difficulty"`
	ClientIP    string `json:"client_ip"`
	UserAgent   string `json:"user_agent"`
	Fingerprint string `json:"fingerprint"`
}

type CreateSemanticCaptchaResponse struct {
	SessionID     string              `json:"session_id"`
	Question      SemanticQuestion     `json:"question"`
	ImageData     string              `json:"image_data"`
	Options       []SemanticOption     `json:"options"`
	ExpiresIn     int64               `json:"expires_in"`
	ExpiresAt     int64               `json:"expires_at"`
}

type SemanticQuestion struct {
	ID          string   `json:"id"`
	Text        string   `json:"text"`
	Hint        string   `json:"hint"`
	Category    string   `json:"category"`
	Difficulty  string   `json:"difficulty"`
	ImageBase64 string   `json:"image_base64,omitempty"`
}

type SemanticOption struct {
	ID    string `json:"id"`
	Text  string `json:"text"`
	Label string `json:"label"`
}

type SemanticCaptchaSession struct {
	SessionID      string            `json:"session_id"`
	QuestionID     string            `json:"question_id"`
	CorrectAnswer  string            `json:"correct_answer"`
	Options        []SemanticOption  `json:"options"`
	Status         string            `json:"status"`
	VerifyCount    int               `json:"verify_count"`
	MaxAttempts    int               `json:"max_attempts"`
	RiskScore      float64           `json:"risk_score"`
	TraceScore     float64           `json:"trace_score"`
	EnvScore       float64           `json:"env_score"`
	CreatedAt      time.Time         `json:"created_at"`
	ExpiredAt      time.Time         `json:"expired_at"`
	ClientIP       string            `json:"client_ip"`
	UserAgent      string            `json:"user_agent"`
	Fingerprint    string            `json:"fingerprint"`
	Language       string            `json:"language"`
	Difficulty     string            `json:"difficulty"`
}

type SemanticQuestionBank struct {
	questions map[string][]SemanticQuestion
}

type SemanticQuestionTemplate struct {
	ID         string   `json:"id"`
	Category   string   `json:"category"`
	Difficulty string   `json:"difficulty"`
	En         QuestionText `json:"en"`
	Zh         QuestionText `json:"zh"`
	Ja         QuestionText `json:"ja"`
	Es         QuestionText `json:"es"`
}

type QuestionText struct {
	Question string   `json:"question"`
	Hint     string   `json:"hint"`
	Options  []OptionText `json:"options"`
	Answer   string   `json:"answer"`
}

type OptionText struct {
	Label string `json:"label"`
	Text  string `json:"text"`
}

func NewSemanticGeneratorService(sessionCache *cache.SessionCache, captchaRepo *db.CaptchaRepository) *SemanticGeneratorService {
	return &SemanticGeneratorService{
		sessionCache: sessionCache,
		captchaRepo:  captchaRepo,
		questionBank: NewSemanticQuestionBank(),
	}
}

func NewSemanticGeneratorServiceSimple() *SemanticGeneratorService {
	return &SemanticGeneratorService{
		questionBank: NewSemanticQuestionBank(),
	}
}

func NewSemanticQuestionBank() *SemanticQuestionBank {
	qb := &SemanticQuestionBank{
		questions: make(map[string][]SemanticQuestion),
	}
	qb.initializeQuestions()
	return qb
}

func (qb *SemanticQuestionBank) initializeQuestions() {
	templates := []SemanticQuestionTemplate{
		{
			ID:         "sem_001",
			Category:   "logic",
			Difficulty: "easy",
			En: QuestionText{
				Question: "What comes next in the sequence: 2, 4, 6, 8, ?",
				Hint:     "Look at the pattern",
				Options: []OptionText{
					{Label: "A", Text: "9"},
					{Label: "B", Text: "10"},
					{Label: "C", Text: "11"},
					{Label: "D", Text: "12"},
				},
				Answer: "B",
			},
			Zh: QuestionText{
				Question: "序列2, 4, 6, 8, ?的下一个是什么？",
				Hint:     "观察规律",
				Options: []OptionText{
					{Label: "A", Text: "9"},
					{Label: "B", Text: "10"},
					{Label: "C", Text: "11"},
					{Label: "D", Text: "12"},
				},
				Answer: "B",
			},
			Ja: QuestionText{
				Question: "数列2, 4, 6, 8, ?の次に来る数は？",
				Hint:     "パターンを見つける",
				Options: []OptionText{
					{Label: "A", Text: "9"},
					{Label: "B", Text: "10"},
					{Label: "C", Text: "11"},
					{Label: "D", Text: "12"},
				},
				Answer: "B",
			},
			Es: QuestionText{
				Question: "¿Qué sigue en la secuencia: 2, 4, 6, 8, ?",
				Hint:     "Observa el patrón",
				Options: []OptionText{
					{Label: "A", Text: "9"},
					{Label: "B", Text: "10"},
					{Label: "C", Text: "11"},
					{Label: "D", Text: "12"},
				},
				Answer: "B",
			},
		},
		{
			ID:         "sem_002",
			Category:   "math",
			Difficulty: "medium",
			En: QuestionText{
				Question: "If all Bloops are Razzles and all Razzles are Lazzles, then all Bloops are definitely:",
				Hint:     "Think about the relationship",
				Options: []OptionText{
					{Label: "A", Text: "Razzles"},
					{Label: "B", Text: "Lazzles"},
					{Label: "C", Text: "Bloopzles"},
					{Label: "D", Text: "Nothing"},
				},
				Answer: "B",
			},
			Zh: QuestionText{
				Question: "如果所有的Bloop都是Razzle，且所有的Razzle都是Lazzle，那么所有的Bloop一定是什么？",
				Hint:     "思考关系",
				Options: []OptionText{
					{Label: "A", Text: "Razzle"},
					{Label: "B", Text: "Lazzle"},
					{Label: "C", Text: "Bloopzle"},
					{Label: "D", Text: "以上都不是"},
				},
				Answer: "B",
			},
			Ja: QuestionText{
				Question: "すべてのBloopがRazzleで、すべてのRazzleがLazzleなら、すべてのBloopは必ず何？",
				Hint:     "関係を考える",
				Options: []OptionText{
					{Label: "A", Text: "Razzle"},
					{Label: "B", Text: "Lazzle"},
					{Label: "C", Text: "Bloopzle"},
					{Label: "D", Text: "何もありません"},
				},
				Answer: "B",
			},
			Es: QuestionText{
				Question: "Si todos los Bloops son Razzles y todos los Razzles son Lazzles, entonces todos los Bloops son definitivamente:",
				Hint:     "Piensa en la relación",
				Options: []OptionText{
					{Label: "A", Text: "Razzles"},
					{Label: "B", Text: "Lazzles"},
					{Label: "C", Text: "Bloopzles"},
					{Label: "D", Text: "Nada"},
				},
				Answer: "B",
			},
		},
		{
			ID:         "sem_003",
			Category:   "pattern",
			Difficulty: "hard",
			En: QuestionText{
				Question: "Which word does NOT belong: Apple, Banana, Carrot, Grape?",
				Hint:     "Think about categories",
				Options: []OptionText{
					{Label: "A", Text: "Apple"},
					{Label: "B", Text: "Banana"},
					{Label: "C", Text: "Carrot"},
					{Label: "D", Text: "Grape"},
				},
				Answer: "C",
			},
			Zh: QuestionText{
				Question: "哪个词不属于这一类：苹果、香蕉、胡萝卜、葡萄？",
				Hint:     "思考分类",
				Options: []OptionText{
					{Label: "A", Text: "苹果"},
					{Label: "B", Text: "香蕉"},
					{Label: "C", Text: "胡萝卜"},
					{Label: "D", Text: "葡萄"},
				},
				Answer: "C",
			},
			Ja: QuestionText{
				Question: "どれが違うか：りんご、バナナ、にんじん、ぶどう？",
				Hint:     "カテゴリーを考える",
				Options: []OptionText{
					{Label: "A", Text: "りんご"},
					{Label: "B", Text: "バナナ"},
					{Label: "C", Text: "にんじん"},
					{Label: "D", Text: "ぶどう"},
				},
				Answer: "C",
			},
			Es: QuestionText{
				Question: "¿Cuál palabra NO pertenece: Manzana, Banana, Zanahoria, Uva?",
				Hint:     "Piensa en las categorías",
				Options: []OptionText{
					{Label: "A", Text: "Manzana"},
					{Label: "B", Text: "Banana"},
					{Label: "C", Text: "Zanahoria"},
					{Label: "D", Text: "Uva"},
				},
				Answer: "C",
			},
		},
		{
			ID:         "sem_004",
			Category:   "spatial",
			Difficulty: "medium",
			En: QuestionText{
				Question: "If you rotate the letter 'N' 180 degrees, what letter does it look like?",
				Hint:     "Visualize the rotation",
				Options: []OptionText{
					{Label: "A", Text: "Z"},
					{Label: "B", Text: "N"},
					{Label: "C", Text: "M"},
					{Label: "D", Text: "W"},
				},
				Answer: "Z",
			},
			Zh: QuestionText{
				Question: "如果将字母'N'旋转180度，它看起来像什么字母？",
				Hint:     "想象旋转后的样子",
				Options: []OptionText{
					{Label: "A", Text: "Z"},
					{Label: "B", Text: "N"},
					{Label: "C", Text: "M"},
					{Label: "D", Text: "W"},
				},
				Answer: "Z",
			},
			Ja: QuestionText{
				Question: "文字'N'を180度回転させると何の文字ようになりますか？",
				Hint:     "回転をイメージする",
				Options: []OptionText{
					{Label: "A", Text: "Z"},
					{Label: "B", Text: "N"},
					{Label: "C", Text: "M"},
					{Label: "D", Text: "W"},
				},
				Answer: "Z",
			},
			Es: QuestionText{
				Question: "Si rotas la letra 'N' 180 grados, ¿qué letra parece?",
				Hint:     "Visualiza la rotación",
				Options: []OptionText{
					{Label: "A", Text: "Z"},
					{Label: "B", Text: "N"},
					{Label: "C", Text: "M"},
					{Label: "D", Text: "W"},
				},
				Answer: "Z",
			},
		},
		{
			ID:         "sem_005",
			Category:   "comparison",
			Difficulty: "easy",
			En: QuestionText{
				Question: "Tom is taller than Jim. Jim is taller than Sam. Who is the shortest?",
				Hint:     "Compare heights",
				Options: []OptionText{
					{Label: "A", Text: "Tom"},
					{Label: "B", Text: "Jim"},
					{Label: "C", Text: "Sam"},
					{Label: "D", Text: "Cannot determine"},
				},
				Answer: "C",
			},
			Zh: QuestionText{
				Question: "Tom比Jim高，Jim比Sam高。谁最矮？",
				Hint:     "比较身高",
				Options: []OptionText{
					{Label: "A", Text: "Tom"},
					{Label: "B", Text: "Jim"},
					{Label: "C", Text: "Sam"},
					{Label: "D", Text: "无法确定"},
				},
				Answer: "C",
			},
			Ja: QuestionText{
				Question: "TomはJimより高い。JimはSamより高い。一番低いのは誰？",
				Hint:     "高さを比較する",
				Options: []OptionText{
					{Label: "A", Text: "Tom"},
					{Label: "B", Text: "Jim"},
					{Label: "C", Text: "Sam"},
					{Label: "D", Text: "判断できない"},
				},
				Answer: "C",
			},
			Es: QuestionText{
				Question: "Tom es más alto que Jim. Jim es más alto que Sam. ¿Quién es el más bajo?",
				Hint:     "Compara las alturas",
				Options: []OptionText{
					{Label: "A", Text: "Tom"},
					{Label: "B", Text: "Jim"},
					{Label: "C", Text: "Sam"},
					{Label: "D", Text: "No se puede determinar"},
				},
				Answer: "C",
			},
		},
		{
			ID:         "sem_006",
			Category:   "arithmetic",
			Difficulty: "hard",
			En: QuestionText{
				Question: "What is 15% of 200?",
				Hint:     "Calculate the percentage",
				Options: []OptionText{
					{Label: "A", Text: "20"},
					{Label: "B", Text: "25"},
					{Label: "C", Text: "30"},
					{Label: "D", Text: "35"},
				},
				Answer: "C",
			},
			Zh: QuestionText{
				Question: "200的15%是多少？",
				Hint:     "计算百分比",
				Options: []OptionText{
					{Label: "A", Text: "20"},
					{Label: "B", Text: "25"},
					{Label: "C", Text: "30"},
					{Label: "D", Text: "35"},
				},
				Answer: "C",
			},
			Ja: QuestionText{
				Question: "200の15%はいくらですか？",
				Hint:     "パーセントを計算する",
				Options: []OptionText{
					{Label: "A", Text: "20"},
					{Label: "B", Text: "25"},
					{Label: "C", Text: "30"},
					{Label: "D", Text: "35"},
				},
				Answer: "C",
			},
			Es: QuestionText{
				Question: "¿Cuánto es el 15% de 200?",
				Hint:     "Calcula el porcentaje",
				Options: []OptionText{
					{Label: "A", Text: "20"},
					{Label: "B", Text: "25"},
					{Label: "C", Text: "30"},
					{Label: "D", Text: "35"},
				},
				Answer: "C",
			},
		},
		{
			ID:         "sem_007",
			Category:   "language",
			Difficulty: "medium",
			En: QuestionText{
				Question: "Which sentence is grammatically correct?",
				Hint:     "Check grammar",
				Options: []OptionText{
					{Label: "A", Text: "She don't like apples"},
					{Label: "B", Text: "She doesn't likes apples"},
					{Label: "C", Text: "She doesn't like apples"},
					{Label: "D", Text: "She not like apples"},
				},
				Answer: "C",
			},
			Zh: QuestionText{
				Question: "哪个句子语法正确？",
				Hint:     "检查语法",
				Options: []OptionText{
					{Label: "A", Text: "她不喜欢苹果"},
					{Label: "B", Text: "她不喜喜欢苹果"},
					{Label: "C", Text: "她不喜欢苹果。"},
					{Label: "D", Text: "她没喜欢苹果"},
				},
				Answer: "C",
			},
			Ja: QuestionText{
				Question: "どの文が文法的に正しいですか？",
				Hint:     "文法をチェック",
				Options: []OptionText{
					{Label: "A", Text: "彼女はリンゴが好きではない"},
					{Label: "B", Text: "彼女はリンゴを好きではない"},
					{Label: "C", Text: "彼女はリンゴが好きではありません"},
					{Label: "D", Text: "彼女はリンゴが好まない"},
				},
				Answer: "C",
			},
			Es: QuestionText{
				Question: "¿Qué oración es gramaticalmente correcta?",
				Hint:     "Revisa la gramática",
				Options: []OptionText{
					{Label: "A", Text: "Ella no le gustan las manzanas"},
					{Label: "B", Text: "Ella no gusta las manzanas"},
					{Label: "C", Text: "Ella no le gusta las manzanas"},
					{Label: "D", Text: "Ella no gusta manzanas"},
				},
				Answer: "C",
			},
		},
		{
			ID:         "sem_008",
			Category:   "time",
			Difficulty: "easy",
			En: QuestionText{
				Question: "If today is Monday, what day will it be in 3 days?",
				Hint:     "Count forward",
				Options: []OptionText{
					{Label: "A", Text: "Tuesday"},
					{Label: "B", Text: "Wednesday"},
					{Label: "C", Text: "Thursday"},
					{Label: "D", Text: "Friday"},
				},
				Answer: "C",
			},
			Zh: QuestionText{
				Question: "如果今天是星期一，3天后是星期几？",
				Hint:     "向前数",
				Options: []OptionText{
					{Label: "A", Text: "星期二"},
					{Label: "B", Text: "星期三"},
					{Label: "C", Text: "星期四"},
					{Label: "D", Text: "星期五"},
				},
				Answer: "C",
			},
			Ja: QuestionText{
				Question: "今日が月曜日なら、3日後は何曜日？",
				Hint:     "前に数える",
				Options: []OptionText{
					{Label: "A", Text: "火曜日"},
					{Label: "B", Text: "水曜日"},
					{Label: "C", Text: "木曜日"},
					{Label: "D", Text: "金曜日"},
				},
				Answer: "C",
			},
			Es: QuestionText{
				Question: "Si hoy es lunes, ¿qué día será en 3 días?",
				Hint:     "Cuenta hacia adelante",
				Options: []OptionText{
					{Label: "A", Text: "Martes"},
					{Label: "B", Text: "Miércoles"},
					{Label: "C", Text: "Jueves"},
					{Label: "D", Text: "Viernes"},
				},
				Answer: "C",
			},
		},
	}

	for _, template := range templates {
		qb.questions[template.Difficulty] = append(qb.questions[template.Difficulty], qb.convertTemplate(template))
		qb.questions["all"] = append(qb.questions["all"], qb.convertTemplate(template))
	}
}

func (qb *SemanticQuestionBank) convertTemplate(template SemanticQuestionTemplate) SemanticQuestion {
	qt := template.En

	return SemanticQuestion{
		ID:         template.ID,
		Text:       qt.Question,
		Hint:       qt.Hint,
		Category:   template.Category,
		Difficulty: template.Difficulty,
	}
}

func (qb *SemanticQuestionBank) GetLocalizedQuestion(templateID, language, difficulty string) SemanticQuestion {
	templates := qb.findTemplateByID(templateID)
	if templates == nil {
		return SemanticQuestion{}
	}

	template := templates[0]
	var qt QuestionText

	switch language {
	case "zh", "zh-CN", "zh-TW":
		qt = template.Zh
	case "ja", "ja-JP":
		qt = template.Ja
	case "es", "es-ES", "es-MX":
		qt = template.Es
	default:
		qt = template.En
	}

	options := make([]SemanticOption, len(qt.Options))
	for i, opt := range qt.Options {
		options[i] = SemanticOption{
			ID:    opt.Label,
			Text:  opt.Text,
			Label: opt.Label,
		}
	}

	shuffledOptions := make([]SemanticOption, len(options))
	copy(shuffledOptions, options)

	for i := len(shuffledOptions) - 1; i > 0; i-- {
		j := rand.Intn(i + 1)
		shuffledOptions[i], shuffledOptions[j] = shuffledOptions[j], shuffledOptions[i]
	}

	return SemanticQuestion{
		ID:         template.ID,
		Text:       qt.Question,
		Hint:       qt.Hint,
		Category:   template.Category,
		Difficulty: template.Difficulty,
	}
}

func (qb *SemanticQuestionBank) findTemplateByID(templateID string) []SemanticQuestionTemplate {
	templates := []SemanticQuestionTemplate{
		{
			ID:         "sem_001",
			Category:   "logic",
			Difficulty: "easy",
			En: QuestionText{
				Question: "What comes next in the sequence: 2, 4, 6, 8, ?",
				Hint:     "Look at the pattern",
				Options: []OptionText{
					{Label: "A", Text: "9"},
					{Label: "B", Text: "10"},
					{Label: "C", Text: "11"},
					{Label: "D", Text: "12"},
				},
				Answer: "B",
			},
			Zh: QuestionText{
				Question: "序列2, 4, 6, 8, ?的下一个是什么？",
				Hint:     "观察规律",
				Options: []OptionText{
					{Label: "A", Text: "9"},
					{Label: "B", Text: "10"},
					{Label: "C", Text: "11"},
					{Label: "D", Text: "12"},
				},
				Answer: "B",
			},
			Ja: QuestionText{
				Question: "数列2, 4, 6, 8, ?の次に来る数は？",
				Hint:     "パターンを見つける",
				Options: []OptionText{
					{Label: "A", Text: "9"},
					{Label: "B", Text: "10"},
					{Label: "C", Text: "11"},
					{Label: "D", Text: "12"},
				},
				Answer: "B",
			},
			Es: QuestionText{
				Question: "¿Qué sigue en la secuencia: 2, 4, 6, 8, ?",
				Hint:     "Observa el patrón",
				Options: []OptionText{
					{Label: "A", Text: "9"},
					{Label: "B", Text: "10"},
					{Label: "C", Text: "11"},
					{Label: "D", Text: "12"},
				},
				Answer: "B",
			},
		},
		{
			ID:         "sem_002",
			Category:   "math",
			Difficulty: "medium",
			En: QuestionText{
				Question: "If all Bloops are Razzles and all Razzles are Lazzles, then all Bloops are definitely:",
				Hint:     "Think about the relationship",
				Options: []OptionText{
					{Label: "A", Text: "Razzles"},
					{Label: "B", Text: "Lazzles"},
					{Label: "C", Text: "Bloopzles"},
					{Label: "D", Text: "Nothing"},
				},
				Answer: "B",
			},
			Zh: QuestionText{
				Question: "如果所有的Bloop都是Razzle，且所有的Razzle都是Lazzle，那么所有的Bloop一定是什么？",
				Hint:     "思考关系",
				Options: []OptionText{
					{Label: "A", Text: "Razzle"},
					{Label: "B", Text: "Lazzle"},
					{Label: "C", Text: "Bloopzle"},
					{Label: "D", Text: "以上都不是"},
				},
				Answer: "B",
			},
			Ja: QuestionText{
				Question: "すべてのBloopがRazzleで、すべてのRazzleがLazzleなら、すべてのBloopは必ず何？",
				Hint:     "関係を考える",
				Options: []OptionText{
					{Label: "A", Text: "Razzle"},
					{Label: "B", Text: "Lazzle"},
					{Label: "C", Text: "Bloopzle"},
					{Label: "D", Text: "何もありません"},
				},
				Answer: "B",
			},
			Es: QuestionText{
				Question: "Si todos los Bloops son Razzles y todos los Razzles son Lazzles, entonces todos los Bloops son definitivamente:",
				Hint:     "Piensa en la relación",
				Options: []OptionText{
					{Label: "A", Text: "Razzles"},
					{Label: "B", Text: "Lazzles"},
					{Label: "C", Text: "Bloopzles"},
					{Label: "D", Text: "Nada"},
				},
				Answer: "B",
			},
		},
		{
			ID:         "sem_003",
			Category:   "pattern",
			Difficulty: "hard",
			En: QuestionText{
				Question: "Which word does NOT belong: Apple, Banana, Carrot, Grape?",
				Hint:     "Think about categories",
				Options: []OptionText{
					{Label: "A", Text: "Apple"},
					{Label: "B", Text: "Banana"},
					{Label: "C", Text: "Carrot"},
					{Label: "D", Text: "Grape"},
				},
				Answer: "C",
			},
			Zh: QuestionText{
				Question: "哪个词不属于这一类：苹果、香蕉、胡萝卜、葡萄？",
				Hint:     "思考分类",
				Options: []OptionText{
					{Label: "A", Text: "苹果"},
					{Label: "B", Text: "香蕉"},
					{Label: "C", Text: "胡萝卜"},
					{Label: "D", Text: "葡萄"},
				},
				Answer: "C",
			},
			Ja: QuestionText{
				Question: "どれが違うか：りんご、バナナ、にんじん、ぶどう？",
				Hint:     "カテゴリーを考える",
				Options: []OptionText{
					{Label: "A", Text: "りんご"},
					{Label: "B", Text: "バナナ"},
					{Label: "C", Text: "にんじん"},
					{Label: "D", Text: "ぶどう"},
				},
				Answer: "C",
			},
			Es: QuestionText{
				Question: "¿Cuál palabra NO pertenece: Manzana, Banana, Zanahoria, Uva?",
				Hint:     "Piensa en las categorías",
				Options: []OptionText{
					{Label: "A", Text: "Manzana"},
					{Label: "B", Text: "Banana"},
					{Label: "C", Text: "Zanahoria"},
					{Label: "D", Text: "Uva"},
				},
				Answer: "C",
			},
		},
		{
			ID:         "sem_004",
			Category:   "spatial",
			Difficulty: "medium",
			En: QuestionText{
				Question: "If you rotate the letter 'N' 180 degrees, what letter does it look like?",
				Hint:     "Visualize the rotation",
				Options: []OptionText{
					{Label: "A", Text: "Z"},
					{Label: "B", Text: "N"},
					{Label: "C", Text: "M"},
					{Label: "D", Text: "W"},
				},
				Answer: "Z",
			},
			Zh: QuestionText{
				Question: "如果将字母'N'旋转180度，它看起来像什么字母？",
				Hint:     "想象旋转后的样子",
				Options: []OptionText{
					{Label: "A", Text: "Z"},
					{Label: "B", Text: "N"},
					{Label: "C", Text: "M"},
					{Label: "D", Text: "W"},
				},
				Answer: "Z",
			},
			Ja: QuestionText{
				Question: "文字'N'を180度回転させると何の文字ようになりますか？",
				Hint:     "回転をイメージする",
				Options: []OptionText{
					{Label: "A", Text: "Z"},
					{Label: "B", Text: "N"},
					{Label: "C", Text: "M"},
					{Label: "D", Text: "W"},
				},
				Answer: "Z",
			},
			Es: QuestionText{
				Question: "Si rotas la letra 'N' 180 grados, ¿qué letra parece?",
				Hint:     "Visualiza la rotación",
				Options: []OptionText{
					{Label: "A", Text: "Z"},
					{Label: "B", Text: "N"},
					{Label: "C", Text: "M"},
					{Label: "D", Text: "W"},
				},
				Answer: "Z",
			},
		},
		{
			ID:         "sem_005",
			Category:   "comparison",
			Difficulty: "easy",
			En: QuestionText{
				Question: "Tom is taller than Jim. Jim is taller than Sam. Who is the shortest?",
				Hint:     "Compare heights",
				Options: []OptionText{
					{Label: "A", Text: "Tom"},
					{Label: "B", Text: "Jim"},
					{Label: "C", Text: "Sam"},
					{Label: "D", Text: "Cannot determine"},
				},
				Answer: "C",
			},
			Zh: QuestionText{
				Question: "Tom比Jim高，Jim比Sam高。谁最矮？",
				Hint:     "比较身高",
				Options: []OptionText{
					{Label: "A", Text: "Tom"},
					{Label: "B", Text: "Jim"},
					{Label: "C", Text: "Sam"},
					{Label: "D", Text: "无法确定"},
				},
				Answer: "C",
			},
			Ja: QuestionText{
				Question: "TomはJimより高い。JimはSamより高い。一番低いのは誰？",
				Hint:     "高さを比較する",
				Options: []OptionText{
					{Label: "A", Text: "Tom"},
					{Label: "B", Text: "Jim"},
					{Label: "C", Text: "Sam"},
					{Label: "D", Text: "判断できない"},
				},
				Answer: "C",
			},
			Es: QuestionText{
				Question: "Tom es más alto que Jim. Jim es más alto que Sam. ¿Quién es el más bajo?",
				Hint:     "Compara las alturas",
				Options: []OptionText{
					{Label: "A", Text: "Tom"},
					{Label: "B", Text: "Jim"},
					{Label: "C", Text: "Sam"},
					{Label: "D", Text: "No se puede determinar"},
				},
				Answer: "C",
			},
		},
		{
			ID:         "sem_006",
			Category:   "arithmetic",
			Difficulty: "hard",
			En: QuestionText{
				Question: "What is 15% of 200?",
				Hint:     "Calculate the percentage",
				Options: []OptionText{
					{Label: "A", Text: "20"},
					{Label: "B", Text: "25"},
					{Label: "C", Text: "30"},
					{Label: "D", Text: "35"},
				},
				Answer: "C",
			},
			Zh: QuestionText{
				Question: "200的15%是多少？",
				Hint:     "计算百分比",
				Options: []OptionText{
					{Label: "A", Text: "20"},
					{Label: "B", Text: "25"},
					{Label: "C", Text: "30"},
					{Label: "D", Text: "35"},
				},
				Answer: "C",
			},
			Ja: QuestionText{
				Question: "200の15%はいくらですか？",
				Hint:     "パーセントを計算する",
				Options: []OptionText{
					{Label: "A", Text: "20"},
					{Label: "B", Text: "25"},
					{Label: "C", Text: "30"},
					{Label: "D", Text: "35"},
				},
				Answer: "C",
			},
			Es: QuestionText{
				Question: "¿Cuánto es el 15% de 200?",
				Hint:     "Calcula el porcentaje",
				Options: []OptionText{
					{Label: "A", Text: "20"},
					{Label: "B", Text: "25"},
					{Label: "C", Text: "30"},
					{Label: "D", Text: "35"},
				},
				Answer: "C",
			},
		},
		{
			ID:         "sem_007",
			Category:   "language",
			Difficulty: "medium",
			En: QuestionText{
				Question: "Which sentence is grammatically correct?",
				Hint:     "Check grammar",
				Options: []OptionText{
					{Label: "A", Text: "She don't like apples"},
					{Label: "B", Text: "She doesn't likes apples"},
					{Label: "C", Text: "She doesn't like apples"},
					{Label: "D", Text: "She not like apples"},
				},
				Answer: "C",
			},
			Zh: QuestionText{
				Question: "哪个句子语法正确？",
				Hint:     "检查语法",
				Options: []OptionText{
					{Label: "A", Text: "她不喜欢苹果"},
					{Label: "B", Text: "她不喜喜欢苹果"},
					{Label: "C", Text: "她不喜欢苹果。"},
					{Label: "D", Text: "她没喜欢苹果"},
				},
				Answer: "C",
			},
			Ja: QuestionText{
				Question: "どの文が文法的に正しいですか？",
				Hint:     "文法をチェック",
				Options: []OptionText{
					{Label: "A", Text: "彼女はリンゴが好きではない"},
					{Label: "B", Text: "彼女はリンゴを好きではない"},
					{Label: "C", Text: "彼女はリンゴが好きではありません"},
					{Label: "D", Text: "彼女はリンゴが好まない"},
				},
				Answer: "C",
			},
			Es: QuestionText{
				Question: "¿Qué oración es gramaticalmente correcta?",
				Hint:     "Revisa la gramática",
				Options: []OptionText{
					{Label: "A", Text: "Ella no le gustan las manzanas"},
					{Label: "B", Text: "Ella no gusta las manzanas"},
					{Label: "C", Text: "Ella no le gusta las manzanas"},
					{Label: "D", Text: "Ella no gusta manzanas"},
				},
				Answer: "C",
			},
		},
		{
			ID:         "sem_008",
			Category:   "time",
			Difficulty: "easy",
			En: QuestionText{
				Question: "If today is Monday, what day will it be in 3 days?",
				Hint:     "Count forward",
				Options: []OptionText{
					{Label: "A", Text: "Tuesday"},
					{Label: "B", Text: "Wednesday"},
					{Label: "C", Text: "Thursday"},
					{Label: "D", Text: "Friday"},
				},
				Answer: "C",
			},
			Zh: QuestionText{
				Question: "如果今天是星期一，3天后是星期几？",
				Hint:     "向前数",
				Options: []OptionText{
					{Label: "A", Text: "星期二"},
					{Label: "B", Text: "星期三"},
					{Label: "C", Text: "星期四"},
					{Label: "D", Text: "星期五"},
				},
				Answer: "C",
			},
			Ja: QuestionText{
				Question: "今日が月曜日なら、3日後は何曜日？",
				Hint:     "前に数える",
				Options: []OptionText{
					{Label: "A", Text: "火曜日"},
					{Label: "B", Text: "水曜日"},
					{Label: "C", Text: "木曜日"},
					{Label: "D", Text: "金曜日"},
				},
				Answer: "C",
			},
			Es: QuestionText{
				Question: "Si hoy es lunes, ¿qué día será en 3 días?",
				Hint:     "Cuenta hacia adelante",
				Options: []OptionText{
					{Label: "A", Text: "Martes"},
					{Label: "B", Text: "Miércoles"},
					{Label: "C", Text: "Jueves"},
					{Label: "D", Text: "Viernes"},
				},
				Answer: "C",
			},
		},
	}

	for _, t := range templates {
		if t.ID == templateID {
			return []SemanticQuestionTemplate{t}
		}
	}
	return nil
}

func (s *SemanticGeneratorService) Create(ctx context.Context, req *CreateSemanticCaptchaRequest) (*CreateSemanticCaptchaResponse, error) {
	sessionID := generateSemanticSessionID()
	expiresAt := time.Now().Add(5 * time.Minute)

	language := req.Language
	if language == "" {
		language = "en"
	}

	difficulty := req.Difficulty
	if difficulty == "" {
		difficulty = "easy"
	}

	templateIDs := []string{"sem_001", "sem_002", "sem_003", "sem_004", "sem_005", "sem_006", "sem_007", "sem_008"}
	selectedTemplateID := templateIDs[rand.Intn(len(templateIDs))]

	question := s.questionBank.GetLocalizedQuestion(selectedTemplateID, language, difficulty)

	templates := s.questionBank.findTemplateByID(selectedTemplateID)
	if templates == nil {
		return nil, fmt.Errorf("failed to find question template")
	}

	template := templates[0]
	var qt QuestionText
	switch language {
	case "zh", "zh-CN", "zh-TW":
		qt = template.Zh
	case "ja", "ja-JP":
		qt = template.Ja
	case "es", "es-ES", "es-MX":
		qt = template.Es
	default:
		qt = template.En
	}

	options := make([]SemanticOption, len(qt.Options))
	for i, opt := range qt.Options {
		options[i] = SemanticOption{
			ID:    opt.Label,
			Text:  opt.Text,
			Label: opt.Label,
		}
	}

	shuffledOptions := make([]SemanticOption, len(options))
	copy(shuffledOptions, options)
	for i := len(shuffledOptions) - 1; i > 0; i-- {
		j := rand.Intn(i + 1)
		shuffledOptions[i], shuffledOptions[j] = shuffledOptions[j], shuffledOptions[i]
	}

	imageBase64 := s.generateQuestionImage(question.Text)

	session := &SemanticCaptchaSession{
		SessionID:     sessionID,
		QuestionID:    selectedTemplateID,
		CorrectAnswer: qt.Answer,
		Options:       shuffledOptions,
		Status:        "pending",
		VerifyCount:   0,
		MaxAttempts:   3,
		RiskScore:     0,
		TraceScore:    0,
		EnvScore:      0,
		CreatedAt:     time.Now(),
		ExpiredAt:     expiresAt,
		ClientIP:      req.ClientIP,
		UserAgent:     req.UserAgent,
		Fingerprint:   req.Fingerprint,
		Language:      language,
		Difficulty:    difficulty,
	}

	if s.sessionCache != nil {
		sessionJSON, err := json.Marshal(session)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal session: %w", err)
		}
		if err := s.sessionCache.SetRaw(ctx, sessionID, string(sessionJSON), 5*time.Minute); err != nil {
			return nil, fmt.Errorf("failed to cache session: %w", err)
		}
	}

	return &CreateSemanticCaptchaResponse{
		SessionID: sessionID,
		Question: SemanticQuestion{
			ID:         question.ID,
			Text:       qt.Question,
			Hint:       qt.Hint,
			Category:   question.Category,
			Difficulty: question.Difficulty,
		},
		ImageData: imageBase64,
		Options:   shuffledOptions,
		ExpiresIn: int64(5 * time.Minute / time.Second),
		ExpiresAt: expiresAt.Unix(),
	}, nil
}

func (s *SemanticGeneratorService) generateQuestionImage(questionText string) string {
	width := 400
	height := 150

	img := image.NewRGBA(image.Rect(0, 0, width, height))

	bgColor := color.RGBA{R: 245, G: 245, B: 245, A: 255}
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, bgColor)
		}
	}

	textColor := color.RGBA{R: 51, G: 51, B: 51, A: 255}
	startX := 30
	startY := height / 2

	fontSize := 16
	if len(questionText) > 50 {
		fontSize = 14
	}

	for i, ch := range questionText {
		x := startX + i*fontSize/2
		if x < width-20 {
			drawSimpleChar(img, x, startY, string(ch), textColor)
		}
	}

	imgBase64 := encodeImageToBase64(img)
	return imgBase64
}

func drawSimpleChar(img *image.RGBA, x, y int, ch string, col color.RGBA) {
	for dy := 0; dy < 12; dy++ {
		for dx := 0; dx < 8; dx++ {
			px := x + dx
			py := y - 6 + dy
			if px >= 0 && px < img.Bounds().Dx() && py >= 0 && py < img.Bounds().Dy() {
				img.Set(px, py, col)
			}
		}
	}
}

func encodeImageToBase64(img *image.RGBA) string {
	imgBase64 := "data:image/png;base64,"
	
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return imgBase64
	}

	encoded := base64.StdEncoding.EncodeToString(buf.Bytes())
	imgBase64 += encoded
	
	return imgBase64
}

func (s *SemanticGeneratorService) GetSession(ctx context.Context, sessionID string) (*SemanticCaptchaSession, error) {
	if s.sessionCache != nil {
		sessionData, err := s.sessionCache.GetRaw(ctx, sessionID)
		if err == nil && sessionData != "" {
			var session SemanticCaptchaSession
			if err := json.Unmarshal([]byte(sessionData), &session); err == nil {
				return &session, nil
			}
		}
	}
	return nil, fmt.Errorf("session not found: %s", sessionID)
}

func (s *SemanticGeneratorService) UpdateSession(ctx context.Context, session *SemanticCaptchaSession) error {
	if s.sessionCache != nil {
		sessionJSON, err := json.Marshal(session)
		if err != nil {
			return fmt.Errorf("failed to marshal session: %w", err)
		}
		remainingTime := time.Until(session.ExpiredAt)
		if remainingTime <= 0 {
			return fmt.Errorf("session expired")
		}
		if err := s.sessionCache.SetRaw(ctx, session.SessionID, string(sessionJSON), remainingTime); err != nil {
			return fmt.Errorf("failed to update session cache: %w", err)
		}
	}
	return nil
}

func generateSemanticSessionID() string {
	return fmt.Sprintf("sem_%d_%d", time.Now().UnixNano(), rand.Intn(10000))
}
