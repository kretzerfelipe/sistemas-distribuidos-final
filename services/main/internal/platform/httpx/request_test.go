package httpx

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type sampleBody struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

func readBody(t *testing.T, body string) (*sampleBody, error) {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	return ReadBody[sampleBody](httptest.NewRecorder(), request)
}

func readVariable[T any](t *testing.T, name, urlPath string) (T, error) {
	t.Helper()
	var value T
	var err error

	router := chi.NewRouter()
	router.Get("/items/{id}", func(_ http.ResponseWriter, r *http.Request) {
		value, err = GetVariable[T](r, name)
	})
	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, urlPath, nil))
	return value, err
}

func TestReadBodyParsesTheJSON(t *testing.T) {
	body, err := readBody(t, `{"name":"Ana","age":30}`)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if body.Name != "Ana" || body.Age != 30 {
		t.Errorf("unexpected body: %+v", body)
	}
}

func TestReadBodyRejectsInvalidBodiesAndReturnsNil(t *testing.T) {
	invalidBodies := map[string]string{
		"empty body":          "",
		"malformed json":      `{"name":`,
		"wrong field type":    `{"name":"Ana","age":"thirty"}`,
		"unknown field":       `{"name":"Ana","role":"ADMIN"}`,
		"two json objects":    `{"name":"Ana"}{"name":"Bob"}`,
		"json array":          `[{"name":"Ana"}]`,
		"trailing characters": `{"name":"Ana"} garbage`,
	}

	for name, invalidBody := range invalidBodies {
		t.Run(name, func(t *testing.T) {
			body, err := readBody(t, invalidBody)

			if !errors.Is(err, ErrInvalidRequest) {
				t.Errorf("want ErrInvalidRequest, got %v", err)
			}
			if body != nil {
				t.Error("body must be nil when there is an error")
			}
		})
	}
}

func TestReadBodyRejectsBodiesAboveTheLimit(t *testing.T) {
	hugeBody := `{"name":"` + strings.Repeat("a", maxBodyBytes) + `"}`

	body, err := readBody(t, hugeBody)

	if !errors.Is(err, ErrInvalidRequest) || body != nil {
		t.Errorf("oversized body must be rejected: body=%v err=%v", body, err)
	}
}

func TestGetVariableReadsAString(t *testing.T) {
	value, err := readVariable[string](t, "id", "/items/abc")

	if err != nil || value != "abc" {
		t.Errorf("want abc, got %q (err=%v)", value, err)
	}
}

func TestGetVariableReadsAnInteger(t *testing.T) {
	value, err := readVariable[int](t, "id", "/items/42")

	if err != nil || value != 42 {
		t.Errorf("want 42, got %d (err=%v)", value, err)
	}
}

func TestGetVariableReadsAnInt64(t *testing.T) {
	value, err := readVariable[int64](t, "id", "/items/9223372036854775807")

	if err != nil || value != 9223372036854775807 {
		t.Errorf("want the max int64, got %d (err=%v)", value, err)
	}
}

func TestGetVariableReadsABoolean(t *testing.T) {
	value, err := readVariable[bool](t, "id", "/items/true")

	if err != nil || !value {
		t.Errorf("want true, got %v (err=%v)", value, err)
	}
}

func TestGetVariableReadsAUUID(t *testing.T) {
	expected := uuid.New()

	value, err := readVariable[uuid.UUID](t, "id", "/items/"+expected.String())

	if err != nil || value != expected {
		t.Errorf("want %s, got %s (err=%v)", expected, value, err)
	}
}

func TestGetVariableRejectsValuesOfTheWrongFormat(t *testing.T) {
	t.Run("integer", func(t *testing.T) {
		value, err := readVariable[int](t, "id", "/items/abc")

		if !errors.Is(err, ErrInvalidRequest) || value != 0 {
			t.Errorf("want ErrInvalidRequest and zero value, got %d / %v", value, err)
		}
	})
	t.Run("uuid", func(t *testing.T) {
		value, err := readVariable[uuid.UUID](t, "id", "/items/not-a-uuid")

		if !errors.Is(err, ErrInvalidRequest) || value != uuid.Nil {
			t.Errorf("want ErrInvalidRequest and zero value, got %s / %v", value, err)
		}
		if !strings.Contains(err.Error(), "id must be a valid UUID") {
			t.Errorf("message should name the variable and the rule, got %q", err.Error())
		}
	})
	t.Run("boolean", func(t *testing.T) {
		_, err := readVariable[bool](t, "id", "/items/maybe")

		if !errors.Is(err, ErrInvalidRequest) {
			t.Errorf("want ErrInvalidRequest, got %v", err)
		}
	})
}

func TestGetVariableRejectsAVariableThatIsNotInTheRoute(t *testing.T) {
	value, err := readVariable[string](t, "other", "/items/abc")

	if !errors.Is(err, ErrInvalidRequest) || value != "" {
		t.Errorf("want ErrInvalidRequest and zero value, got %q / %v", value, err)
	}
}

func TestGetVariableTreatsAnUnsupportedTypeAsAProgrammingError(t *testing.T) {
	_, err := readVariable[float64](t, "id", "/items/1.5")

	if err == nil {
		t.Fatal("an unsupported type must fail")
	}
	if errors.Is(err, ErrInvalidRequest) {
		t.Error("an unsupported type is a bug, not a client mistake: it must not look like a 400")
	}
}
