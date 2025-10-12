package user

type Repo interface {
	GetAll() ([]*User, error)
	GetByID(id string) (*User, error)
	Create(u *User) error
	CheckEmailExists(email string) (bool, error)
	GetByEmail(email string) (*User, error)
}
