package models

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"lq-center-go/consts"
	"lq-center-go/utils/cache"
	"regexp"
	"strconv"
	"time"

	"github.com/astaxie/beego/logs"
	"github.com/astaxie/beego/orm"
	"github.com/astaxie/beego/validation"
)

// User Model Struct
type User struct {
	Id            int64
	Account       string `orm:"size(40)"`
	Password      string `orm:"size(200)"`
	Channel       string
	LastLoginIP   string       `orm:"null;size(50)"`
	LastLoginTime time.Time    `orm:"null;type(datetime)"`
	Profile       *UserProfile `orm:"null;rel(one)"`
	CreatedTime   time.Time    `orm:"auto_now_add;type(datetime)"`
	UpdatedTime   time.Time    `orm:"auto_now;type(datetime)"`
}

// UserProfile Mode Struct
type UserProfile struct {
	Id          int64
	UserId      int64     `orm:"unique"`
	Name        string    `orm:"size(100)"`
	Sex         byte      `orm:"default(0)"`
	Phone       string    `orm:"size(20);unique"`
	Email       string    `orm:"size(50);unique"`
	Address     string    `orm:"size(100)"`
	CreatedTime time.Time `orm:"auto_now_add;type(datetime)"`
	UpdatedTime time.Time `orm:"auto_now;type(datetime)"`
}

// UserInput 输入
type UserInput struct {
	Account  string `valid:"Required; MaxSize(40); MinSize(5)"`
	Password string `valid:"Required; MaxSize(40); MinSize(5)"`
	Channel  string `valid:"Required"`
}

// RegisterInput 注册输入
type RegisterInput struct {
	Account  string `valid:"Required; MaxSize(40); MinSize(5)"`
	Password string `valid:"Required; MaxSize(40); MinSize(5)"`
	Channel  string `valid:"Required"`
	Type     string
}

// TokenOutput token返回
type TokenOutput struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    time.Duration
	CreateAt     time.Time
	User         interface{}
}

// UserUpdateInput 用户信息编辑
type UserUpdateInput struct {
	Id      int64
	Name    string `orm:"size(100)"`
	Sex     byte
	Phone   string `orm:"size(20)"`
	Email   string `orm:"size(50)"`
	Address string `orm:"size(100)"`
}

func init() {
	// register model
	orm.RegisterModel(new(User), new(UserProfile))
}

// User2ProfileOutput
func User2ProfileOutput(user *User) interface{} {
	out := map[string]interface{}{
		"id":            user.Id,
		"account":       user.Account,
		"lastLoginIP":   user.LastLoginIP,
		"lastLoginTime": user.LastLoginTime.Format(consts.TimeFormatString)}
	if user.Profile != nil {
		out["name"] = user.Profile.Name
		out["sex"] = user.Profile.Name
		out["phone"] = user.Profile.Phone
		out["email"] = user.Profile.Email
		out["address"] = user.Profile.Address
		out["createdTime"] = user.Profile.CreatedTime.Format(consts.TimeFormatString)
		out["updatedTime"] = user.Profile.UpdatedTime.Format(consts.TimeFormatString)
	}
	return out
}

// CheckAccountValid 检查账号密码是否有效
func CheckAccountValid(value string) error {
	if value == "" {
		return ErrAccountPassNull
	}

	validAccount := validation.Validation{}
	validAccount.Match(value, regexp.MustCompile(`^[a-zA-Z0-9_-]*$`), "")

	validEmail := validation.Validation{}
	validEmail.Email(value, "邮箱")

	validPhone := validation.Validation{}
	validPhone.Phone(value, "电话")

	if validAccount.HasErrors() && validEmail.HasErrors() && validPhone.HasErrors() {
		return ErrAccountPassInvalid
	}

	return nil
}

// Register  注册
func Register(userInput *RegisterInput) (*User, error) {
	if userInput == nil {
		return nil, ErrFailed
	}
	if userInput.Account == "" {
		return nil, ErrAccountPassNull
	}
	if userInput.Channel == "" {
		return nil, ErrFailed
	}
	o := orm.NewOrm()
	var user User
	user.Account = userInput.Account

	rdErr := o.Read(&user, "account")
	// 检查此账号是否已存在
	if rdErr == orm.ErrNoRows {
		err := o.Begin()
		if err != nil {
			return nil, ErrRegisterFailed
		}
		profile := UserProfile{}

		user.Password = passwordEncode(userInput.Password)
		user.Channel = userInput.Channel
		user.Profile = &profile

		_, err2 := o.Insert(&profile)
		if err2 != nil {
			fmt.Println(err2)
			o.Rollback()
			return nil, ErrRegisterFailed
		}

		uid, err2 := o.Insert(&user)
		if err2 != nil {
			fmt.Println(err2)
			o.Rollback()
			return nil, ErrRegisterFailed
		}

		profile.UserId = uid
		_, err2 = o.Update(&profile, "UserId")
		if err2 != nil {
			fmt.Println(err2)
			o.Rollback()
			return nil, ErrRegisterFailed
		}

		retUser := User{Id: uid}
		err2 = o.Read(&retUser)
		// fmt.Println(err2)

		if err2 == nil {
			err = o.Commit()
			if err == nil {
				return &retUser, nil
			}
		}
		fmt.Println(err2)
		o.Rollback()
		return nil, ErrRegisterFailed
	}
	return nil, ErrAccountExsit
}

// Login
// 登陆
func Login(userInput *UserInput) (*User, error) {
	if userInput == nil {
		return nil, ErrFailed
	}
	if userInput.Account == "" {
		return nil, ErrAccountPassNull
	}
	if userInput.Channel == "" {
		return nil, ErrFailed
	}

	valid := validation.Validation{}
	b, err := valid.Valid(userInput)
	if err != nil {
		return nil, ErrAccountPassInvalid
	}
	if !b {
		return nil, ErrFailed
	}

	// 检查账号
	accountErr := CheckAccountValid(userInput.Account)
	if accountErr != nil {
		return nil, accountErr
	}

	// 检查密码
	passErr := CheckAccountValid(userInput.Password)
	if passErr != nil {
		return nil, passErr
	}

	o := orm.NewOrm()
	newPassword := passwordEncode(userInput.Password)
	user := User{Account: userInput.Account, Password: newPassword}

	qs := o.QueryTable("user")
	rdErr := qs.Filter("Account", userInput.Account).Filter("Password", newPassword).One(&user)
	// rdErr := o.Read(&user)
	if rdErr == nil {
		LoginSuccess(user.Id, "")

		return &user, nil
	}

	return nil, ErrAccountPass
}

// LoginSuccess 登陆成功
func LoginSuccess(id int64, ip string) {
	if id == 0 || ip == "" {
		return
	}

	o := orm.NewOrm()
	user := User{Id: id}

	if o.Read(&user) == nil {
		user.LastLoginIP = ip
		user.LastLoginTime = time.Now()
		if num, err := o.Update(&user); err == nil {
			fmt.Println(num)
		}
	}
}

// GetUserProfile 获取用户信息
func GetUserProfile(id int64) (*User, error) {
	o := orm.NewOrm()
	var user User
	user.Id = id

	err := o.Read(&user)
	if err == nil {
		return &user, nil
	}

	return nil, ErrUserNotExsit
}

// UpdateUserProfile 修改用户信息
func UpdateUserProfile(input *UserUpdateInput) (int64, error) {
	o := orm.NewOrm()
	var user User
	user.Id = input.Id
	profile := UserProfile{UserId: input.Id}
	profile.Name = input.Name
	profile.Phone = input.Phone
	profile.Address = input.Address
	profile.Sex = input.Sex
	profile.Email = input.Email
	user.Profile = &profile

	num, err := o.Update(user)
	// err := o.Read(&user)
	if err == nil {
		return num, nil
	}
	return 0, ErrUserUpdateFailed
}

// ResetUserPasswordToken 获取重置密码token
func ResetUserPasswordToken(account string, redirectUri string) (*User, string, error) {
	if cache.RDCache == nil {
		return nil, "", ErrFailed
	}

	// 清理旧的token
	if cache.RDCache.IsExist(account) {
		logs.Debug("清理Token")
		t := string(cache.RDCache.Get(account).([]byte))
		cache.RDCache.Delete(account)
		cache.RDCache.Delete(t)
	}

	jv, err := json.Marshal(map[string]interface{}{
		"account": account,
		"uri":     redirectUri,
	})
	if err != nil {
		logs.Error(err)
		return nil, "", ErrFailed
	}

	o := orm.NewOrm()
	user := User{Account: account}

	if o.Read(&user, "account") == nil {
		if user.Profile != nil {
			if o.Read(user.Profile) != nil {
				return nil, "", ErrEmailNotExsit
			}
		}
		if user.Profile != nil && user.Profile.Email != "" {
			ha256 := sha256.New()
			ha256.Write([]byte(account))
			ha256.Write([]byte(strconv.FormatInt(time.Now().UnixNano(), 10)))
			hashedToken := ha256.Sum(nil)

			token := hex.EncodeToString(hashedToken)
			err := cache.RDCache.Put(account, token, consts.ResetPasswordTokenTime)
			logs.Error(err)
			err = cache.RDCache.Put(token, jv, consts.ResetPasswordTokenTime)
			// err = cache.RDCache.Put(token, account, consts.ResetPasswordTokenTime)
			logs.Error(err)
			// logs.Info(string(cache.RDCache.Get(account).([]byte])))

			return &user, token, nil
		} else {
			return nil, "", ErrEmailNotExsit
		}
	}

	return nil, "", ErrAccountNotExsit
}

// ResetUserPassword 重置密码
func ResetUserPassword(token string, password string) (map[string]interface{}, error) {
	if cache.RDCache == nil {
		return nil, ErrFailed
	}

	_t := map[string]interface{}{
		"account": "",
		"uri":     "",
	}

	jv := cache.RDCache.Get(token)
	if jv == nil {
		return nil, ErrTokenExpired
	}

	err := json.Unmarshal(jv.([]byte), &_t)
	if err != nil {
		logs.Error(err)
		return nil, ErrFailed
	}

	account := _t["account"]
	if account == nil {
		return nil, ErrTokenExpired
	}

	o := orm.NewOrm()
	user := User{Account: account.(string)}

	if o.Read(&user, "account") == nil {
		user.Password = passwordEncode(password)
		_, err := o.Update(&user, "password")
		if err != nil {
			logs.Error(err)
			return nil, ErrResetPwFailed
		}
		return _t, nil
	}

	return nil, ErrAccountNotExsit
}

func passwordEncode(password string) string {
	ha256 := sha256.New()
	ha256.Write([]byte(password))
	hashedPass := ha256.Sum(nil)

	return hex.EncodeToString(hashedPass)
}
