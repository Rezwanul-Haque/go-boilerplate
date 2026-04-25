package tasks

const TypeExampleTask = "example:task"

type ExamplePayload struct {
	UserID  string `json:"user_id"`
	Message string `json:"message"`
}
