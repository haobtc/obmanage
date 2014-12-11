package controllers

import "obManage/app/models"
import "github.com/revel/revel"
import "github.com/revmgo"
import "time"
import "crypto/rand"
import "net/smtp"
import "strconv"
import "fmt"
import  "labix.org/v2/mgo"
import "labix.org/v2/mgo/bson"
func init() {
	go watchAuthReset()
}
func watchAuthReset() {
	fmt.Printf("init watch\n")
	revel.Config.SetSection("prod")
	revurl,_ := revel.Config.String("revmgo.dial")
	if (revurl=="") {
		revurl = "mongodb://localhost:27017/richwallet"
	}

	session, err := mgo.Dial(revurl)
	if err!=nil {
		panic(err.Error())
	}
	//var it *Iter
	for {
		time.Sleep(time.Duration(30) * time.Second)
		handleReset(session)
	}
}

func handleReset(session *mgo.Session) {
	//只保留一条最近的记录
	//
	var result models.RequestCancelAuthCs
	iter := session.DB("richwallet").C("reset").Find(bson.M{"timeout":false, "handled":false, "sentemail":false }).Iter()
	revel.Config.SetSection("mail")
	afterM, _ := revel.Config.Int("mail.times")
	tunit, _ := revel.Config.Int("mail.unit")
	
	if afterM ==0 {
		afterM = 5
	}

	if tunit == 0 {
		tunit = 24*60*60
	}
	for iter.Next(&result) {
		println("send mail check to " + result.Email)
		email := result.Email
		lang := result.Lang
		go sendmailNeedCheck(email, result.Code, result.Lang)
		go notifyAdminCancelAuth(email)
		models.SetSentEmail(session, email, true)
	
		go func() {
			time.Sleep(time.Duration(afterM *tunit) * time.Second)
			println("set timeout" + email)
			err := models.SetAuthTimeout(session, email, true)
			if err!=nil {
				println(err.Error())
			}
		}()

		go func(){
			for i:=0; i< afterM; i++ {
				time.Sleep(time.Duration(1 * tunit) * time.Second);
				if i+1 !=afterM {
					if models.IsAuthVerified(session, email) {
						go sendmailRemindReset(email , true, time.Now().Add(time.Duration(tunit)*time.Second), afterM-i-1 , lang)
					} else {
						go sendmailNeedVerify(email, lang);
					}
				} else {
					if models.IsAuthVerified(session, email) {
						models.DeleteAuth(session, email)
						println("delete auth +" + email)
						go sendmailResetSuccess(email, time.Now().Add(time.Duration(tunit)*time.Second), lang)
						go notifyAdminCancelAuthSuccess(email)
						models.SetAuthHandled(session, email, true)
					} else {
						//models.SetAuthHandled(session, email, true)
						//not verified ,nothin todo
					}
				}

			}
		}()
		
		if err := iter.Close(); err != nil {
			println(err.Error())
		}
	}
}

type OBAuthManage struct {
	*revel.Controller
	revmgo.MongoController
}

type Response struct {
	Success bool
	StatusCode int
	Info interface{}
}

func (c OBAuthManage) Index() revel.Result {
	return c.Render()
}

func (c OBAuthManage) Login(email string, pwd string) revel.Result {
	var passed = models.AdminExists(c.MongoSession, email, pwd);
	if passed {
		c.Session["user"] = email;
		return c.RenderJson(Response{Info:"login", Success:true})
	} else {
		return c.RenderJson(Response{Info: "login", Success: false})
	}
}

func (c OBAuthManage) Admin(email string) revel.Result {
	connected, curAdmin := c.connected();
	if !connected {
		return c.RenderTemplate("OBAuthManage/needAuth.html");
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

func (c OBAuthManage) connected() (bool, string) {
	if username, ok := c.Session["user"]; ok {
		return true , username;
	}
	return false, "";
}

func (c OBAuthManage) Customers(flags string, pages string, frompages string) revel.Result {
	connected, _ := c.connected();
	if !connected {
		return c.RenderTemplate("OBAuthManage/needAuth.html");
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


func (c OBAuthManage)Search(email string) revel.Result {
	connected, _ := c.connected();
	if !connected {
		return c.RenderTemplate("OBAuthManage/needAuth.html");
	}
	
	customers := models.SearchCustomers(c.MongoSession, email)
	c.RenderArgs["customers"] = customers;
	customerCount := len(customers);
	c.RenderArgs["customerCount"] = customerCount
	return c.RenderTemplate("OBAuthManage/customers.html")
}

func (c OBAuthManage) AuthCustomers() revel.Result {
	connected, _ := c.connected();
	if !connected {
		return c.RenderTemplate("OBAuthManage/needAuth.html");
	}
	
	authCustomers := models.AuthUser(c.MongoSession)
	authCount := len(authCustomers)
	c.RenderArgs["authCount"] = authCount;
	c.RenderArgs["authCustomers"] = authCustomers;
	return c.Render(authCustomers);
}

func (c OBAuthManage) ResetPetitioners() revel.Result {
	connected, _ := c.connected();
	if !connected {
		return c.RenderTemplate("OBAuthManage/needAuth.html");
	}

	requestCancelAuth := models.AllResetPetitioners(c.MongoSession)
	count := len(requestCancelAuth)
	c.RenderArgs["Count"] = count;
	c.RenderArgs["CancelCus"] = requestCancelAuth
	return c.RenderTemplate("OBAuthManage/authCustomers.html");
}

func (c OBAuthManage)DeleteAuth(customer string) revel.Result {
	connected, _ := c.connected();
	if !connected {
		return c.RenderTemplate("OBAuthManage/needAuth.html");
	}
	
	cus := models.DeleteAuth(c.MongoSession, customer);
	c.RenderArgs["authCount"] = len(cus)
	c.RenderArgs["authCustomers"] = cus

	return c.RenderTemplate("app/authCustomers.html")
}

func (c OBAuthManage)AddAdmin(email string, mpass string) revel.Result {
	connected, _ := c.connected();
	if !connected {
		return c.RenderTemplate("OBAuthManage/needAuth.html");
	}
	
	if (models.AddAdmin(c.MongoSession, email, mpass)) {
		var m map[string]string = make(map[string]string)
		m["create"] = email;
		m["pass"] = mpass;
		
		return c.RenderJson(Response{Info:m});
	} else {
		return c.RenderJson(Response{Info:"user exists"});
	}
}

func (c OBAuthManage)DeleteAdmin(admin string) revel.Result {
	connected, curAdmin := c.connected();
	if !connected || !models.AdminRoot(c.MongoSession, curAdmin){
		return c.RenderTemplate("OBAuthManage/needAuth.html");
	}

	admins, deleted :=models.DeleteAdmin(c.MongoSession, admin)
	c.RenderArgs["Admins"] = admins
	c.RenderArgs["AdminCount"] = len(admins)
	c.RenderArgs["ShowWhenRoot"] = true;
	c.RenderArgs["EditSuccess"] = deleted;
	return	c.RenderTemplate("OBAuthManage/admin.html")
}

func (c OBAuthManage)EditAdminEmail(srcemail string, dstemail string) revel.Result {
	connected, _ := c.connected();
	if !connected {
		return c.RenderTemplate("OBAuthManage/needAuth.html");
	}
	c.Request.ParseForm();
	dstemail = (c.Request.Form["value"][0]);
	_,err := models.EditAdminEmail(c.MongoSession, srcemail, dstemail);
	if err!= nil {
		return c.RenderJson(Response{Info: srcemail})
		panic(err.Error())
	} else {
		return c.RenderJson(Response{Info: srcemail})
	}
}

func (c OBAuthManage)EditAdminPass(srcemail string, newpass string) revel.Result {
	connected, _ := c.connected();
	if !connected {
		return c.RenderTemplate("OBAuthManage/needAuth.html");
	}
	c.Request.ParseForm();

	newpass = (c.Request.Form["value"][0]);
	_,err := models.EditAdminPass(c.MongoSession, srcemail, newpass);
	if err!= nil {
		return c.RenderJson(Response{Info: srcemail})
	} else {
		return c.RenderJson(Response{Info: srcemail})
	}
}

func (c OBAuthManage)EditAuth(srcemail string, authKey string) revel.Result {
	connected, _ := c.connected();
	if !connected {
		return c.RenderTemplate("OBAuthManage/needAuth.html");
	}
	c.Request.ParseForm();
	authKey = (c.Request.Form["value"][0]);

	_,err := models.EditAuth(c.MongoSession, srcemail, authKey);
	if err!= nil {
		return c.RenderJson(Response{Info: srcemail})
	} else {
		return c.RenderJson(Response{Info: srcemail})
	}
}

func (c OBAuthManage)RequestCancelAuth(email string) revel.Result {
	connected, _ := c.connected();
	if !connected {
		return c.RenderTemplate("OBAuthManage/needAuth.html");
	}	
	
//cs := models.SearchAuthCustomers(c.MongoSession, email) 
//if len(cs) == 0 {
//	return c.RenderJson(Response{Info: "no such user"});
//}

//err := models.AddRequestResetAuth(c.MongoSession, email)
//var retmsg = ""
//if err!=nil {
//	retmsg = err.Error()
//}

	requestCancelAuth := models.AllResetPetitioners(c.MongoSession)
	count := len(requestCancelAuth)
	c.RenderArgs["Count"] = count;
	c.RenderArgs["CancelCus"] = requestCancelAuth
	c.RenderArgs["error"] = "";
	return c.RenderTemplate("OBAuthManage/authCustomers.html");
}

func (c OBAuthManage)SentEmail(email string, value bool) revel.Result {
	err := models.SetSentEmail(c.MongoSession, email, value);
	retmsg := "ok"
	if err!=nil {
		retmsg=err.Error()
	}
	return c.RenderJson(Response{Info:retmsg})
}

func (c OBAuthManage)AuthValidateUser(email string, value bool) revel.Result {
	err := models.SetAuthValidate(c.MongoSession, email, value);
	retmsg := "ok"
	if err!=nil {
		retmsg=err.Error()
	}
	return c.RenderJson(Response{Info:retmsg})
}

func (c OBAuthManage)AuthFrozenAccount(email string, value bool) revel.Result {
	err := models.SetAuthFrozenAccount(c.MongoSession, email, value);
	retmsg := "ok"
	if err!=nil {
		retmsg=err.Error()
	}
	return c.RenderJson(Response{Info:retmsg})
}

func (c OBAuthManage)AuthHandled(email string, value bool) revel.Result {
	err := models.SetAuthHandled(c.MongoSession, email, value);
	retmsg := "ok"
	if err!=nil {
		retmsg=err.Error()
	}
	return c.RenderJson(Response{Info:retmsg})
}

func (c OBAuthManage)AuthTimeout(email string, value bool) revel.Result {
	err := models.SetAuthTimeout(c.MongoSession, email, value);
	retmsg := "ok"
	if err!=nil {
		retmsg=err.Error()
	}
	return c.RenderJson(Response{Info:retmsg})
}

func sendmail(to []string, sub []byte, content []byte) bool {

	revel.Config.SetSection("mail")
	user,_ := revel.Config.String("mail.user")
	passwd,_ := revel.Config.String("mail.passwd")
	host,_ := revel.Config.String("mail.host")
	from,_ := revel.Config.String("mail.from")
	auth := smtp.PlainAuth(
		"",
		user,//email
		passwd,
		host,//host
	)
	body := append(sub, content...)
	err := smtp.SendMail(host+":25", auth, from, to, body)
	if err!=nil {
		println("send mail fails")
		println(err.Error())
		return true
	}
	return false
}

func sendmailResetSuccess(email string, cancelTime time.Time,  lang string) {
	var sub, msg []byte
	if lang=="zh-cn" {
		sub = []byte("subject:用户取消两步验证\r\n\r\n")
		msg = []byte("您于 " + cancelTime.String() + " 申请取消两步认证已经生效")
	} else {
		sub = []byte("subject:reset google code\r\n")
		msg = []byte("You requested to reset google auth at " + cancelTime.String() + ", we have reset your google auth. You may set a new google auth")
	}

	sendmail([]string{email}, sub, msg)
}



func notifyAdminCancelAuth(email string) {
	sub := []byte("subject:用户取消两步验证\r\n\r\n")
	msg := []byte("用户: " + email + " 取消两步认证")
	sendmail([]string{email}, sub, msg)
}

func notifyAdminCheckCodeSuccess(email string){
	sub := []byte("subject:用户取消两步验证码通过\r\n\r\n")
	msg := []byte("用户: " + email + " 取消两步认证通过")
	sendmail([]string{email}, sub, msg)
}

func notifyAdminCancelAuthSuccess(email string) {
	sub := []byte("subject:用户成功取消两步验证码\r\n\r\n")
	msg := []byte("用户: " + email + " 成功取消两步认证")
	sendmail([]string{email}, sub, msg)
}

func sendmailNeedCheck(email string, code string, lang string) {
	var sub, msg []byte;
	if lang=="zh-cn" {
		sub = []byte("subject:取消两步验证\r\n\r\n")
		msg = []byte("您的验证码是: " + code)
	} else {
		sub = []byte("subject:reset google auth\r\n\r\n")
		msg = []byte("Your verify code : " + code)
	}
	sendmail([]string{email}, sub, msg)
}

func sendmailNeedVerify(email string, lang string) {
	//code := randNumberHex(6);
	var sub, msg []byte;
	if lang=="zh-cn" {
		sub = []byte("subject:取消两步验证\r\n\r\n")
		msg = []byte("您提交申请取消两步验证, 请及时输入验证码确认")
	} else {
		sub = []byte("subject:reset google auth\r\n\r\n")
		msg = []byte("You request to reset google auth, please submit your verify code form an earlier emai")
	}
	sendmail([]string{email}, sub, msg)
}

func sendmailRemindReset(email string, verified bool, t time.Time, afterDays int, lang string) {
	var sub, msg []byte;
	if lang=="zh-cn" {
		sub = []byte("subject:取消两步验证\r\n\r\n" )
		if verified {
			msg = []byte("您于 " + t.String() + " 申请取消两步认证,我们将在" + strconv.Itoa(afterDays)+"日后重置您的两步验证")
		} else {
			msg = []byte("您于 " + t.String() + " 申请取消两步认证,请提交您的验证码, 通过后我们将在" + strconv.Itoa(afterDays)+"日后重置您的两步验证")
		}
	} else {
		sub = []byte("subject:reset google code\r\n")
		if verified {
			msg = []byte("You requested to reset google auth at " + t.String() + ". We will reset your google auth in " + strconv.Itoa(afterDays) + " days"	)
		} else {
			msg = []byte("You requested to reset google auth at " + t.String() + ". After your verify, we will reset your google auth in " + strconv.Itoa(afterDays) + " days"	)
		}
}

	sendmail([]string{email}, sub, msg)
}

func sendmailAlreadyRequested(email string, t time.Time, lang string) {
	var sub, msg []byte
	sub = []byte("subject:取消两步验证\r\n\r\n")
	msg = []byte("用户: " + email + " 您在" + t.String() + "已经提交申请取消二步验证, 本次提交将忽略")
	sendmail([]string{email}, sub, msg)
}

//request cancel auth
//unc (c OBAuthManage) RequestResetAuth () revel.Result {
//	revel.Config.SetSection("mail")
//	afterM, _ := revel.Config.Int("mail.times")
//	tunit, _ := revel.Config.Int("mail.unit")
//	
//	if afterM ==0 {
//		afterM = 5
//	}
//
//	if tunit == 0 {
//		tunit = 24*60*60
//	}
//
//	c.Request.ParseForm()
//	fmt.Printf("%v",c.Request.PostFormValue("email"))
//	email := c.Params.Form.Get("email");
//	serverKey := c.Params.Form.Get("serverKey")
//	lang := c.Params.Form.Get("lang");
//	
//	if(email=="" || serverKey=="") {
//		c.RenderJson(Response{Success:false, Info:"User not exists or passwords error"})
//	}
//	
//	if !models.IsAuthUserExists(c.MongoSession, email, serverKey) {
//		return c.RenderJson(Response{Success:false, Info:"User not exists or passwords error or auth not exists"})
//	}	
//
//	code := randNumberHex(6);
//	_, exists := models.AddRequestResetAuth(c.MongoSession, email, code); 
//	if (exists) {
//		
//		go sendmailAlreadyRequested(email, ReqTime(c.MongoSession, email), lang)
//		return  c.RenderJson(Response{Success:false, Info:"Already requested before"})
//	}
//	go notifyAdminCancelAuth(email)
//	go sendmailNeedCheck(email, lang, code);
//	revel.Config.SetSection("prod")
//	revurl,_ := revel.Config.String("revmgo.dial")
//	if (revurl=="") {
//		revurl = "mongodb://localhost:27017/test"
//	}
//	
//	go func() {
//		time.Sleep(time.Duration(afterM *tunit) * time.Second)
//		session, err := mgo.Dial(revurl)
//		if err!=nil {
//			println(err.Error())
//		}
//		defer session.Close()
//		err = models.SetAuthTimeout(session, email, true)
//		if err!=nil {
//			println(err.Error())
//		}
//	}()
//	go func(){
//		session, err := mgo.Dial(revurl)
//		if err!=nil {
//			panic(err.Error())
//		}
//		defer session.Close()
//
//		for i:=0; i< afterM; i++ {
//			time.Sleep(time.Duration(1 * tunit) * time.Second);
//			if i+1 !=afterM {
//				if models.IsAuthVerified(session, email) {
//					go sendmailRemindReset(email , true, time.Now().Add(time.Duration(tunit)*time.Second), afterM-i-1 , lang)
//				} else {
//					go sendmailNeedVerify(email, lang);
//				}
//			}
//
//			if models.IsAuthVerified(session, email) {
//				models.DeleteAuth(session, email)
//				go sendmailResetSuccess(email, time.Now().Add(time.Duration(tunit)*time.Second), lang)
//				go notifyAdminCancelAuthSuccess(email)
//				models.SetAuthHandled(session, email, true)
//			} else {
//				//not verified ,nothin todo
//			}
//		}
//	}()
//	return c.RenderJson(Response{Success:true});
//

func (c OBAuthManage)VerifyResetAuthCode() revel.Result {
	c.Request.ParseForm()
	email := c.Params.Form.Get("email");
	serverKey := c.Params.Form.Get("serverKey")
	code := c.Params.Form.Get("code")
	if (email=="" || serverKey=="" || code=="") {
		c.RenderJson(Response{Success:false, Info:"User not exists or passwords error or auth code invalid"})
	}
	if !models.IsAuthUserExists(c.MongoSession, email, serverKey) {
		return c.RenderJson(Response{Info:"User not exists or passwords error or auth not exists", Success: false})
	}
	if models.VerifyResetAuthCode(c.MongoSession, email, serverKey, code) {
		go notifyAdminCheckCodeSuccess(email)
		return c.RenderJson(Response{Success:true, Info:"5"})
	} else {
		return c.RenderJson(Response{Info:"Verify code is invalid", Success: false})
	}
}

func randNumberHex(size int) string {
	b := make([]byte, size/2)
	rand.Read(b)
	var hex string
	for _,v := range b {
		hex += fmt.Sprintf("%x",v);
	}
	if len(hex) == size-1 {
		hex += "0"
	}
	return hex
}
