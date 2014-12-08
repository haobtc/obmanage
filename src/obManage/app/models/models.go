
package models

import (
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
	"time"
	"errors"
)


type Admin struct{
	Id  bson.ObjectId `bson:"_id,omitempty"`
	Email string `bson:"email"`
	Pass  string  `bson:"pass"`
	IsRoot bool `bson:"isroot"`
}

type Customer struct {
	Id  bson.ObjectId `bson:"_id,omitempty"`
	Action string `bson:"action"`
	Addresses []string `bson:"addresses"`
	AuthKey string `bson:"authKey"`
	Email string `bson:"email"`
	Lang string `bson:"lang"`
	PayloadHash string `bson:"payloadhash"`
	ServerKey string `bson:"serverKey"`
	Wallet map[string]string `bson:"wallet"`
}

type RequestCancelAuthCs struct {
	Id  bson.ObjectId `bson:"_id,omitempty"`
	Email string `bson:"email"`
	ReqTime time.Time `bson:"reqTime"`
	EmailChecked bool `bson:"checked"`
	FrozenAccount bool `bson:"frozened"`
	Timeout   bool `bson:"timeout"`
	Handled bool    `bson:"handled"`
	SentEmail bool  `bson:"sentEmail"`
	Code string `bson:"code"`

}

func AdminExists(s *mgo.Session, email string, pwd string) bool {
	var coll =  s.DB("admin").C("admin");
	num,err :=  coll.Find(bson.M{"email": email, "pass": pwd}).Count();
	if err!=nil {
		println(err.Error())
	}
	if num == 1 {
		return true;
	}
	return false;
}

func AdminRoot(s *mgo.Session, email string) bool {
	var coll = s.DB("admin").C("admin");
	num,err :=  coll.Find(bson.M{"email": email, "isroot":true}).Count();
	if err!=nil {
		println(err.Error())
	}
	if num== 1 {
		return true;
	}
	return false;
}
func Collection(s *mgo.Session) *mgo.Collection {
	return s.DB("").C("")
}

func AllAdmins(s *mgo.Session) []Admin {
	var coll = s.DB("admin").C("admin");
	var admins []Admin
	coll.Find(nil).All(&admins)
	return admins;
}

func CustomersCount(s *mgo.Session) int  {
	n,e := s.DB("richwallet").C("wallet").Count()
	if e!=nil {
		println(e.Error())
	}
	return n;
}

func AuthUser(s *mgo.Session) []Customer {
	var customers []Customer;
	s.DB("richwallet").C("wallet").Find(bson.M{"authKey":bson.M{"$exists":true}}).All(&customers)
	return customers
}

func AllCustomers(s *mgo.Session) []Customer {
	var customers []Customer;
	s.DB("richwallet").C("wallet").Find(nil).All(&customers)
	return customers
}

func CustomerPage(s *mgo.Session, frompage int ,pagesize int)[]Customer {
	var customers []Customer;
	s.DB("richwallet").C("wallet").Find(nil).Skip(frompage*pagesize).
		Limit(pagesize).All(&customers);
	return customers
}

func DeleteAuth(s *mgo.Session, email string) ([]Customer){
	colQuerier := bson.M{"email": email}
	unset := bson.M{"authKey": bson.M{"$exists":true}}
	change := bson.M{"$unset": unset}
	err := s.DB("richwallet").C("wallet").Update(colQuerier, change)
	if err != nil {
		panic(err)
	}
	var customer []Customer
	s.DB("richwallet").C("wallet").Find(bson.M{"authKey":bson.M{"$exists":true}}).All(&customer)
	return customer
}

func SearchCustomers(s *mgo.Session, email string) []Customer {
	var customers []Customer;
	s.DB("richwallet").C("wallet").Find(bson.M{"email":email}).All(&customers)
	return customers
}

func SearchAuthCustomers(s *mgo.Session, email string) []Customer {
	var customers []Customer;
	s.DB("richwallet").C("wallet").Find(bson.M{"email":email, "authKey":bson.M{"$exists":true}}).All(&customers)
	return customers
}

func AddAdmin(s *mgo.Session, email string, pass string) bool {
	count,_ := s.DB("admin").C("admin").Find(bson.M{"email":email}).Count()
	
	if count>0 {
		println("create duplicate")
		return false;
	}
	s.DB("admin").C("admin").Insert(bson.M{"email": email, "pass": pass});
	return true;
}

func DeleteAdmin(s *mgo.Session, email string) ([]Admin, bool) {
	count,_ := s.DB("admin").C("admin").Find(bson.M{"email":email, "isroot":true}).Count()
	var deleted = false;
	if count>0 {
		panic("root only have one");
	} else {
		s.DB("admin").C("admin").Remove(bson.M{"email": email});
		deleted = true
	}
	var admins []Admin;
	s.DB("admin").C("admin").Find(nil).All(&admins);
	return admins, deleted
}

func EditAdminEmail(s *mgo.Session, srcemail string, dstemail string) (string ,error) {
	count,_ := s.DB("admin").C("admin").Find(bson.M{"email":dstemail}).Count()
	if count > 0 {
		return "",errors.New(" dst email exists");
	}
	
	err := s.DB("admin").C("admin").Update(bson.M{"email":srcemail}, bson.M{"$set": bson.M{"email":dstemail}});
	if err!=nil {
		panic(err.Error())
	}
	return dstemail, err
}

func EditAdminPass(s *mgo.Session, srcemail string, newpass string) (string ,error) {
	count,_ := s.DB("admin").C("admin").Find(bson.M{"email":srcemail}).Count()
	if count == 0 {
		return "",errors.New("email  doesnot exists");
	}
	
	err := s.DB("admin").C("admin").Update(bson.M{"email":srcemail}, bson.M{"$set": bson.M{"pass": newpass}});
	if err!=nil {
		panic(err.Error())
	}
	return newpass, err
}

/*************************************************************
 * 功能--    
 * 场景-- 
 * 依赖--
 **************************************************************/
func EditAuth(s *mgo.Session, srcemail string, authKey string) (string ,error) {
	count,_ := s.DB("richwallet").C("wallet").Find(bson.M{"email":srcemail}).Count()
	if count == 0 {
		return "",errors.New("email  doesnot exists");
	}

	var err error;
	if authKey == "" {
		DeleteAuth(s, srcemail);
	} else {
		err = s.DB("richwallet").C("wallet").Update(bson.M{"email":srcemail}, bson.M{"$set": bson.M{"authKey": authKey}});
		if err!=nil {
			println(err.Error())
		}
	}
	return authKey, err
}

func AllResetPetitioners(s *mgo.Session) []RequestCancelAuthCs{
	var all []RequestCancelAuthCs 
	err := s.DB("cancel").C("cancel").Find(nil).All(&all)
	if err!=nil {
		println(err.Error())
	}
	return all
}

func AddRequestResetAuth(s *mgo.Session, email string, code string) (error, bool) {
/*************************************************************
 * 功能-- add the user who requested to reset auth
 * 场景-- 用户发送请求重置 Auth
 * 依赖--
 * 逻辑-- 判断是否曾经提交过, 判断上次提交是否已经超时
 **************************************************************/

	var  result RequestCancelAuthCs;
	query := s.DB("cancel").C("cancel").Find(bson.M{"email": email})
	err := query.Sort("-reqTime").One(&result)
	if err!=nil {
		return s.DB("cancel").C("cancel").Insert(bson.M{"email":email, 
			"reqTime":time.Now(), 
			"checked": false,
			"frozened": false,
			"handled": false,
			"timeout": false,
			"sentEmail": false,
			"code": code}), false
	}
	
	if result.Timeout {
		println("update")
		return s.DB("cancel").C("cancel").Update(bson.M{"email":email},bson.M{"$set": bson.M{"reqTime":time.Now(), "checked": false,"frozened": false,					"handled": false,"timeout": false,"sentEmail": false, "code": code}}) , false
	} else {
		println("has requested before")
		return errors.New("update fails"), true;
	}
}

func SetSentEmail(s *mgo.Session, email string, value bool) error {
	return s.DB("cancel").C("cancel").Update(bson.M{"email":email, "sentEmail":!value}, bson.M{"$set": bson.M{"sentEmail":value}})
}

func SetAuthValidate(s *mgo.Session, email string, value bool) error {
	return s.DB("cancel").C("cancel").Update(bson.M{"email":email, "checked":!value}, bson.M{"$set": bson.M{"checked":value}})
}

func SetAuthFrozenAccount(s *mgo.Session, email string, value bool) error {
	return s.DB("cancel").C("cancel").Update(bson.M{"email":email, "frozened":!value}, bson.M{"$set": bson.M{"frozened":value}})
}

func SetAuthHandled(s *mgo.Session, email string, value bool) error {
	return s.DB("cancel").C("cancel").Update(bson.M{"email":email, "handled":!value}, bson.M{"$set": bson.M{"handled":value}})
}

func SetAuthTimeout(s *mgo.Session, email string, value bool) error {
	return s.DB("cancel").C("cancel").Update(bson.M{"email":email, "timeout":!value}, bson.M{"$set": bson.M{"timeout":value}})
}

func IsAuthUserExists(s *mgo.Session, email string, serverKey string) bool {
	var coll =  s.DB("richwallet").C("wallet");
	num,err :=  coll.Find(bson.M{"email": email, "serverKey":serverKey, "authKey": bson.M{"$exists" : true} }).Count();
	if err!=nil {
		println(err.Error())
	}
	if num >= 1 {
		return true;
	}
	return false;
}


func AddVerfiedCode(s *mgo.Session, email string, serverKey string, code string) bool {
/*************************************************************
 * 功能--
 * 场景--
 * 依赖--
 * 逻辑--
 **************************************************************/
	var coll =  s.DB("admin").C("admin");
	num,err :=  coll.Find(bson.M{"email": email, "serverKey":serverKey, "authKey": bson.M{"exists" : true} }).Count();
	if err!=nil {
		println(err.Error())
		return false
	}
	
	if  num>0 {
		coll = s.DB("cancel").C("cancel");
		err = s.DB("cancel").C("cancel").Update(bson.M{"email":email}, bson.M{"$set": bson.M{"code":code,"checked":false}})
		if err!=nil {
			println(err.Error())
			return false
		}
	}

	return false;
}

func IsAuthVerified(s *mgo.Session, email string) bool {
	var coll =  s.DB("cancel").C("cancel");
	var result  RequestCancelAuthCs;
	err := coll.Find(bson.M{"email": email}).One(&result)
	if  err!=nil {
		println(err.Error())
		return false
	} 
	return result.EmailChecked
}

func VerifyResetAuthCode(s *mgo.Session, email string, serverKey string, code string) bool {
	var result RequestCancelAuthCs
	println(email)
	println(len(email))
	println(code)
	query := s.DB("cancel").C("cancel").Find(bson.M{"email": email})
	err := query.Sort("-reqTime").One(&result)
	if err!= nil {
		
		println(err.Error())
		return false;
	}
	
	println(result.Code)
	if result.Code == code && result.Code!="" {
		println("----------compare")
		s.DB("cancel").C("cancel").Update(bson.M{"email":email}, bson.M{"$set": bson.M{ "checked": true}})
		return true
	}
	return false;
}
