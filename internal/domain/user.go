package domain

type User struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Password string `json:"-"`
	Role     string `json:"role"`
}

type Permission struct {
	UserID          string `json:"userId"`
	ServerID        string `json:"serverId"`
	CanViewConsole  bool   `json:"canViewConsole"`
	CanControlPower bool   `json:"canControlPower"`
}

type PublicLink struct {
	Token    string `json:"token"`
	ServerID string `json:"serverId"`
	Action   string `json:"action"`
}
