package request

import "encoding/json"

type UpdateSettings struct {
	Values map[string]json.RawMessage `json:"values" label:"设置项" validate:"required,min=1"`
}
