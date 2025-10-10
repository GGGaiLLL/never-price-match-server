package user

// Repo 定义用户仓储接口，放在 user 包以避免循环依赖
type Repo interface {
	GetAll() ([]*User, error)
	GetByID(id string) (*User, error)
	Create(u *User) error
	CheckEmailExists(email string) (bool, error) // 新增
}
