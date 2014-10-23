package controllers

import "obManage/app/models"
import "github.com/revel/revel"
import "github.com/revmgo"
//import "strings"
//import "fmt"
import "strconv"
//import "code.google.com/p/go.crypto/bcrypt"



type App struct {
	*revel.Controller
	revmgo.MongoController
}

type Res struct {
	Ok bool
	Code int
	Msg string
	Id string
	List interface{}
	Item interface{}
}

func NewRes() Res {
	return Res{Ok: false}
}

func (c App) Index() revel.Result {
	return c.Render()
}

func (c App) Login(email string, pwd string) revel.Result {
	var passed = models.AdminExists(c.MongoSession, email, pwd);
	if passed {
		c.Session["user"] = email;
		return c.RenderJson(Res{Msg:"login", Ok:true})
	} else {
		return c.RenderJson(Res{Msg: "login", Ok: false})
	}
}

//unc (c App) SaveUser(user models.Admins, verifyPass string) bool {
//	c.Validation.Required(verifyPass)
//	c.Validation.Required(verifyPass == user.Pwd).
//		Message("Password does not match")
//	user.Validate(c.Validation)
//	
//	if c.Validation.HasErrors() {
//		c.Validation.Keep()
//		c.FlashParams()
//		return false;
//	}
//
//	user.HashedPassword, _ = bcrypt.GenerateFromPassword(
//		[]byte(user.Password), bcrypt.DefaultCost)
//	err := c.Txn.Insert(&user)
//	if err != nil {
//		panic(err)
//	}
//
//	c.Session["user"] = user.email
//	c.Flash.Success("Welcome, " + user.email)
//	 return true;
//

func (c App) Admin(email string) revel.Result {
	connected, curAdmin := c.connected();
	if !connected {
		return c.RenderTemplate("App/needAuth.html");
	}
	c.RenderArgs["Admin"] = curAdmin;

	if models.AdminRoot(c.MongoSession, curAdmin) {
		allAdmins := models.AllAdmins(c.MongoSession);
		c.RenderArgs["Admins"] = allAdmins
		c.RenderArgs["AdminCount"] = len(allAdmins)
		c.RenderArgs["ShowWhenRoot"] = true;
	} else {
		c.RenderArgs["ShowWhenRoot"] = false;
	}

	return c.Render();
}

func (c App) connected() (bool, string) {
	if username, ok := c.Session["user"]; ok {
		return true , username;
	}
	
	return false, "";
}

func (c App) Customers(flags string, pages string, frompages string ) revel.Result {
	connected, _ := c.connected();
	if !connected {
		return c.RenderTemplate("App/needAuth.html");
	}
	var total = models.CustomersCount(c.MongoSession)
	var frompage = 0 ;
	var pagesize = 10;
	var prev = false;
	var err error;
	frompage ,err =  strconv.Atoi(frompages)
	if err!=nil {
		//		panic(err.Error())
		frompage = 0;
	}
	pagesize, err = strconv.Atoi(pages)
	if err!=nil {
		pagesize = 10;
	}
	
	if(flags=="prev") {
		prev = true;
	}

	if prev {
		if frompage <= 1 {
			frompage = 0
		} else {
			frompage -= 2
		}
	} else {
		if frompage > total/pagesize {
			frompage = total/pagesize
		}
	}
	
	cs := models.CustomerPage(c.MongoSession, frompage, pagesize);
	c.RenderArgs["customerCount"] = total;
	c.RenderArgs["customers"] = cs;
	var showpage = frompage
	if len(cs)!=0{
		showpage += 1
	}
	c.RenderArgs["Page"] = strconv.Itoa(showpage);
	return c.Render()
}


func (c App)Search(email string) revel.Result {
	connected, _ := c.connected();
	if !connected {
		return c.RenderTemplate("App/needAuth.html");
	}
	
	customers := models.SearchCustomers(c.MongoSession, email)
	c.RenderArgs["customers"] = customers;
	customerCount := len(customers);
	c.RenderArgs["customerCount"] = customerCount
	return c.RenderTemplate("App/customers.html")
}

func (c App) AuthCustomers() revel.Result {
	connected, _ := c.connected();
	if !connected {
		return c.RenderTemplate("App/needAuth.html");
	}
	
	authCustomers := models.AuthUser(c.MongoSession)
	authCount := len(authCustomers)
	c.RenderArgs["authCount"] = authCount;
	c.RenderArgs["authCustomers"] = authCustomers;
	return c.Render(authCustomers);
}

func (c App)DeleteAuth(customer string) revel.Result {
	
	connected, _ := c.connected();
	if !connected {
		return c.RenderTemplate("App/needAuth.html");
	}
	
	cus := models.DeleteAuth(c.MongoSession, customer);
	c.RenderArgs["authCount"] = len(cus)
	c.RenderArgs["authCustomers"] = cus
	return c.RenderTemplate("app/authCustomers.html")
}



func (c App)CreateAdmin(email string, mpass string) revel.Result {
	connected, _ := c.connected();
	if !connected {
		return c.RenderTemplate("App/needAuth.html");
	}
	
	if (models.CreateAdmin(c.MongoSession, email, mpass)) {
		var m map[string]string = make(map[string]string)
		m["create"] = email;
		m["pass"] = mpass;
		return c.RenderJson(Res{Item:m});
	} else {
		return c.RenderJson(Res{Msg:"user exists"});
	}
}

func (c App)DeleteAdmin(admin string) revel.Result {
	connected, curAdmin := c.connected();
	if !connected || !models.AdminRoot(c.MongoSession, curAdmin){
		return c.RenderTemplate("App/needAuth.html");
	}

	admins, deleted :=models.DeleteAdmin(c.MongoSession, admin)
	c.RenderArgs["Admins"] = admins
	c.RenderArgs["AdminCount"] = len(admins)
	c.RenderArgs["ShowWhenRoot"] = true;
	c.RenderArgs["EditSuccess"] = deleted;
	return	c.RenderTemplate("App/admin.html")
}

func (c App)EditAdminEmail(srcemail string, dstemail string) revel.Result {
	connected, _ := c.connected();
	if !connected {
		return c.RenderTemplate("App/needAuth.html");
	}
	c.Request.ParseForm();
//	if dstemail=="" {
//		dstemail = c.Request.Form["value"];
//	}
	dstemail = (c.Request.Form["value"][0]);
	_,err := models.EditAdminEmail(c.MongoSession, srcemail, dstemail);
	if err!= nil {
		return c.RenderJson(Res{Msg: srcemail})
		panic(err.Error())
	} else {
		return c.RenderJson(Res{Msg: srcemail})
	}
	
}

func (c App)EditAdminPass(srcemail string, newpass string) revel.Result {
	connected, _ := c.connected();
	if !connected {
		return c.RenderTemplate("App/needAuth.html");
	}
	c.Request.ParseForm();

	newpass = (c.Request.Form["value"][0]);
	_,err := models.EditAdminPass(c.MongoSession, srcemail, newpass);
	if err!= nil {
		return c.RenderJson(Res{Msg: srcemail})
	} else {
		return c.RenderJson(Res{Msg: srcemail})
	}
}

func (c App)EditAuth(srcemail string, authKey string) revel.Result {
	connected, _ := c.connected();
	if !connected {
		return c.RenderTemplate("App/needAuth.html");
	}
	println("calling edit auth model")
	c.Request.ParseForm();
	authKey = (c.Request.Form["value"][0]);

	_,err := models.EditAuth(c.MongoSession, srcemail, authKey);
	if err!= nil {
		return c.RenderJson(Res{Msg: srcemail})
	} else {
		return c.RenderJson(Res{Msg: srcemail})
	}
}
