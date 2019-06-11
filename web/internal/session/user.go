package session

type User struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Mobile        string `json:"mobile"`
	Email         string `json:"email"`
	Priv          string `json:"priv"`
	LastLoginDate string `json:"last_login_date"`
	LoginCount    int    `json:"login_count"`
}
