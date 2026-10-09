package database

type User struct {
	ID        int64  `json:"id" bson:"_id"`
	Username  string `json:"username" bson:"username"`
	FirstName string `json:"first_name" bson:"first_name"`
	JoinedAt  int64  `json:"joined_at" bson:"joined_at"`
}

type BannedUser struct {
	ID       int64  `json:"id" bson:"_id"`
	Reason   string `json:"reason" bson:"reason"`
	BannedAt int64  `json:"banned_at" bson:"banned_at"`
}

type Database interface {
	AddUser(id int64, username, firstName string) error
	GetUsers() ([]int64, error)
	CountUsers() (int64, error)
	IsBanned(id int64) bool
	BanUser(id int64, reason string) error
	UnbanUser(id int64) error
	IncrementRenames() error
	GetTotalRenames() int64
	Close() error
}

func NewDatabase(mongoURI, dbName string) (Database, error) {
	if mongoURI != "" {
		return NewMongoDatabase(mongoURI, dbName)
	}
	return NewMemoryDatabase(), nil
}
