package httpx

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"futhub/main/internal/platform/util"
)

const maxBodyBytes = 1 << 20

var ErrInvalidRequest = util.NewBusinessError("invalid request")

func ReadBody[T any](w http.ResponseWriter, r *http.Request) (*T, error) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	var body T
	err := decoder.Decode(&body)
	if err != nil {
		return nil, fmt.Errorf("%w: body must be a valid JSON object with only the expected fields", ErrInvalidRequest)
	}

	bodyHasTrailingData := !errors.Is(decoder.Decode(&struct{}{}), io.EOF)
	if bodyHasTrailingData {
		return nil, fmt.Errorf("%w: body must contain a single JSON object", ErrInvalidRequest)
	}
	return &body, nil
}

func GetVariable[T any](r *http.Request, name string) (T, error) {
	var value T

	rawValue := chi.URLParam(r, name)
	variableIsMissing := rawValue == ""
	if variableIsMissing {
		return value, invalidVariable(name, "is required")
	}

	switch target := any(&value).(type) {
	case *string:
		*target = rawValue
	case *int:
		number, err := strconv.Atoi(rawValue)
		if err != nil {
			return value, invalidVariable(name, "must be an integer")
		}
		*target = number
	case *int64:
		number, err := strconv.ParseInt(rawValue, 10, 64)
		if err != nil {
			return value, invalidVariable(name, "must be an integer")
		}
		*target = number
	case *bool:
		flag, err := strconv.ParseBool(rawValue)
		if err != nil {
			return value, invalidVariable(name, "must be true or false")
		}
		*target = flag
	case *uuid.UUID:
		id, err := uuid.Parse(rawValue)
		if err != nil {
			return value, invalidVariable(name, "must be a valid UUID")
		}
		*target = id
	default:
		return value, fmt.Errorf("unsupported type %T for url variable %q", value, name)
	}
	return value, nil
}

func invalidVariable(name, expectation string) error {
	return fmt.Errorf("%w: %s %s", ErrInvalidRequest, name, expectation)
}
