package session

import (
	"crypto/md5"
	"encoding/hex"
	"github.com/Nixson/db"
	"github.com/Nixson/logger"
	"gorm.io/gorm"
)

type User struct {
	Id       uint64 `gorm:"primarykey" json:"id"`
	Access   uint   `json:"access"`
	Login    string `gorm:"index" json:"login"`
	Username string `json:"username"`
	Password string `json:"-"`
}

func (u User) TableName() string {
	return "user"
}

var sql *gorm.DB

func init() {
	db.AfterInit(func() {
		sql = db.Get().Table("user")
		err := sql.AutoMigrate(User{})
		if err != nil {
			logger.Fatal(err)
		}
	})
}

func HashPassword(text string) string {
	hasher := md5.New()
	hasher.Write([]byte(text))
	return hex.EncodeToString(hasher.Sum(nil))
}

func UserRm(id uint64) {
	sql.Delete(User{Id: id})
}
func UserCreate(usr User) {
	sql.Create(&usr)
}
func UserModify(usr User) {
	var oldUsr User
	sql.First(&oldUsr, User{Id: usr.Id})
	if usr.Password == "" {
		usr.Password = oldUsr.Password
	}
	if usr.Access == 0 {
		usr.Access = oldUsr.Access
	}
	sql.Save(&usr)
}

func GetUserById(id uint64) User {
	var usr User
	sql.First(&usr, User{Id: id})
	return usr
}
func GetUsers() []User {
	var usr []User
	sql.Find(&usr)
	return usr
}
func GetUserLogin(login string) User {
	var usr User
	sql.First(&usr, User{Login: login})
	return usr
}
func GetUserByLoginAndPassword(login string, pass string) User {
	var usr User
	sql.First(&usr, User{Login: login, Password: HashPassword(pass)})
	return usr
}
