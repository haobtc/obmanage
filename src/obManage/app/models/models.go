package models

import (
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
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
	println("num");
	println(num);
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
	println(len(customers))
	return customers
}

func AllCustomers(s *mgo.Session) []Customer {
	var customers []Customer;
	s.DB("richwallet").C("wallet").Find(nil).All(&customers)
	return customers
}

func CustomerPage(s *mgo.Session, frompage int ,pagesize int)[]Customer {
	var customers []Customer;
	println("skip")
	println(frompage * pagesize)
	s.DB("richwallet").C("wallet").Find(nil).Skip(frompage*pagesize).
		Limit(pagesize).All(&customers);
	return customers
}

func DeleteAuth(s *mgo.Session, email string) ([]Customer){
	println(email)
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

func CreateAdmin(s *mgo.Session, email string, pass string) bool {
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
		println("is root");
	} else {
		s.DB("admin").C("admin").Remove(bson.M{"email": email});
		deleted = true
	}
	var admins []Admin;
	
	s.DB("admin").C("admin").Find(nil).All(&admins);
	return admins, deleted
}

func EditAdminEmail(s *mgo.Session, srcemail string, dstemail string) (string ,error) {
	println(srcemail)
	println(dstemail)
	
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

func EditAuth(s *mgo.Session, srcemail string, authKey string) (string ,error) {
	println("edit auth in")
	count,_ := s.DB("richwallet").C("wallet").Find(bson.M{"email":srcemail}).Count()
	if count == 0 {
		println("err occur")
		return "",errors.New("email  doesnot exists");
	}
	println("authkey ==="+authKey);
	println("email ===" + srcemail);
	var err error;
	if authKey == "" {
		println("to --- unset")
		DeleteAuth(s, srcemail);
		println(srcemail)
	} else {
		err = s.DB("richwallet").C("wallet").Update(bson.M{"email":srcemail}, bson.M{"$set": bson.M{"authKey": authKey}});
		if err!=nil {
			println(err.Error())
		}
	}
	return authKey, err
}
