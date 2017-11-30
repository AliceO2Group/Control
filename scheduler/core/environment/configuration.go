package environment

import(
)
type Configuration struct {
	Flps []Role		`json:"flps" binding:"required"`
}