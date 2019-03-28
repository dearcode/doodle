package rbac

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/dearcode/crab/http/client"
	"github.com/dearcode/crab/log"
	"github.com/juju/errors"

	"github.com/dearcode/doodle/rbac/meta"
)

func (c Client) get(url string) ([]byte, error) {
	return client.New().Timeout(c.timeout).Get(url, map[string]string{"Token": c.token}, nil)
}

//Post do post request.
func (c Client) post(url string, body []byte) ([]byte, error) {
	return client.New().Timeout(c.timeout).Post(url, map[string]string{"Token": c.token, "Content-Type": "application/x-www-form-urlencoded"}, body)
}

//Put do pust request.
func (c Client) put(url string, body []byte) ([]byte, error) {
	return client.New().Timeout(c.timeout).Put(url, map[string]string{"Token": c.token}, body)
}

//Delete do delete request.
func (c Client) delete(url string) ([]byte, error) {
	return client.New().Timeout(c.timeout).Delete(url, map[string]string{"Token": c.token}, nil)
}

const (
	timeout = 10
)

//Client rbac 客户端.
type Client struct {
	host    string
	token   string
	timeout int
}

type rbacResponse struct {
	Status  int
	Message string
	Data    int64
}

//New 创建rbac客户端.
func New(host, token string) *Client {
	return &Client{
		timeout: timeout,
		token:   token,
		host:    host,
	}
}

func (c Client) responseError(buf []byte) error {
	resp := rbacResponse{}
	if err := json.Unmarshal(buf, &resp); err != nil {
		log.Infof("Unmarshal response error:%v, resp:%v", err, string(buf))
		return errors.Trace(err)
	}

	if resp.Status != 0 {
		return fmt.Errorf(resp.Message)
	}

	return nil
}

func (c Client) responseID(buf []byte) (int64, error) {
	resp := rbacResponse{}
	if err := json.Unmarshal(buf, &resp); err != nil {
		log.Infof("Unmarshal response error:%v, resp:%v", err, string(buf))
		return 0, errors.Trace(err)
	}

	if resp.Status != 0 {
		return 0, fmt.Errorf(resp.Message)
	}

	return resp.Data, nil
}

//GetUserResources 根据用户邮箱，获取关联的资源.
func (c Client) GetUserResources(email string) ([]meta.Resource, error) {
	url := fmt.Sprintf("http://%s/rbac/user/resource/?email=%s", c.host, email)
	buf, err := c.get(url)
	if err != nil {
		log.Infof("Get:%v error:%v", url, err)
		return nil, errors.Annotatef(err, "url:%v", url)
	}

	resp := struct {
		Status  int
		Data    []meta.Resource
		Message string
	}{}

	if err = json.Unmarshal(buf, &resp); err != nil {
		log.Infof("Unmarshal error:%v, buf:%v", err, string(buf))
		return nil, errors.Annotatef(err, "url:%v", url)
	}

	if resp.Status != 0 {
		return nil, errors.Errorf(resp.Message)
	}

	return resp.Data, nil
}

//GetUserResourceIDs 根据用户邮箱，获取关联的资源ID.
func (c Client) GetUserResourceIDs(email string) ([]int64, error) {
	res, err := c.GetUserResources(email)
	if err != nil {
		return nil, err
	}

	var ids []int64

	for _, re := range res {
		ids = append(ids, re.ID)
	}

	return ids, nil
}

//GetUser 获取用户信息.
func (c Client) GetUser(email string) (meta.User, error) {
	url := fmt.Sprintf("http://%s/rbac/user/info/?email=%s", c.host, email)

	buf, err := c.get(url)
	if err != nil {
		log.Infof("Get:%v error:%v", url, err)
		return meta.User{}, errors.Annotatef(err, "url:%v", url)
	}

	log.Infof("get user info resp:%v", string(buf))

	resp := struct {
		Status  int
		Data    meta.User
		Message string
	}{}

	if err = json.Unmarshal(buf, &resp); err != nil {
		return meta.User{}, errors.Trace(err)
	}

	if resp.Status != 0 {
		return meta.User{}, errors.New(resp.Message)
	}

	return resp.Data, nil
}

//PostResource 添加资源.
func (c Client) PostResource(name, comments string) (int64, error) {
	form := url.Values{}
	form.Add("Name", name)
	form.Add("Comments", comments)
	form.Encode()

	log.Debugf("name:%v, comment:%v", name, comments)

	url := fmt.Sprintf("http://%s/rbac/resource/", c.host)

	buf, err := c.post(url, []byte(form.Encode()))
	if err != nil {
		log.Infof("Post:%v error:%v", url, err)
		return 0, errors.Trace(err)
	}

	log.Infof("add resource resp:%v", string(buf))

	return c.responseID(buf)
}

//PutRole 修改角色信息.
func (c Client) PutRole(roleID int64, name, comments string) error {
	form := url.Values{}
	form.Add("role_id", fmt.Sprintf("%d", roleID))
	form.Add("name", name)
	form.Add("comments", comments)
	form.Encode()

	log.Debugf("id:%v name:%v, comment:%v", roleID, name, comments)

	url := fmt.Sprintf("http://%s/rbac/role/", c.host)

	buf, err := c.put(url, []byte(form.Encode()))
	if err != nil {
		log.Infof("Put:%v error:%v", url, err)
		return errors.Trace(err)
	}

	log.Infof("modify role:%v resp:%s", roleID, buf)

	return c.responseError(buf)
}

//PostRole 添加角色.
func (c Client) PostRole(name, comments, user, email string) (int64, error) {
	form := url.Values{}
	form.Add("name", name)
	form.Add("comments", comments)
	form.Add("user", user)
	form.Add("email", email)
	form.Encode()

	log.Debugf("name:%v, comment:%v", name, comments)

	url := fmt.Sprintf("http://%s/rbac/role/", c.host)

	buf, err := c.post(url, []byte(form.Encode()))
	if err != nil {
		log.Infof("Post:%v error:%v", url, err)
		return 0, errors.Trace(err)
	}

	log.Infof("add role resp:%v", string(buf))

	return c.responseID(buf)
}

//PostRoleResource 关联角色与资源.
func (c Client) PostRoleResource(roleID, resID int64) (int64, error) {
	url := fmt.Sprintf("http://%s/rbac/role/resource/?role_id=%d&resource_id=%d", c.host, roleID, resID)
	buf, err := c.post(url, nil)
	if err != nil {
		log.Infof("Post:%v error:%v", url, err)
		return 0, errors.Trace(err)
	}

	log.Infof("add role resource resp:%v", string(buf))

	return c.responseID(buf)
}

//DeleteResourceRole 删除资源与角色对应关系.
func (c Client) DeleteResourceRole(resID, roleID int64) error {
	url := fmt.Sprintf("http://%s/rbac/role/resource/?role_id=%d&resource_id=%d", c.host, roleID, resID)
	buf, err := c.delete(url)
	if err != nil {
		log.Infof("Delete:%v error:%v", url, err)
		return errors.Trace(err)
	}

	log.Infof("resource:%v role:%v response:%s", resID, roleID, buf)

	_, err = c.responseID(buf)
	return err
}

//DeleteResource 删除资源.
func (c Client) DeleteResource(resID int64) error {
	url := fmt.Sprintf("http://%s/rbac/resource/?id=%d", c.host, resID)
	buf, err := c.delete(url)
	if err != nil {
		log.Infof("Delete:%v error:%v", url, err)
		return errors.Trace(err)
	}
	log.Infof("delete resource resp:%v", string(buf))

	return c.responseError(buf)
}

//DeleteRoleUser 删除角色中用户.
func (c Client) DeleteRoleUser(roleID int64, email string) error {
	url := fmt.Sprintf("http://%s/rbac/role/user/?role_id=%d&email=%s", c.host, roleID, email)
	buf, err := c.delete(url)
	if err != nil {
		log.Infof("Delete role user error:%v", err)
		return errors.Trace(err)
	}

	log.Infof("delete role:%v user:%v, resp:%v", roleID, email, string(buf))

	return c.responseError(buf)
}

//DeleteRole 删除角色，根据名称或者ID.
func (c Client) DeleteRole(id int64, name string) error {
	url := fmt.Sprintf("http://%s/rbac/role/?id=%d&name=%s", c.host, id, name)
	buf, err := c.delete(url)
	if err != nil {
		log.Infof("Delete:%v error:%v", url, err)
		return errors.Trace(err)
	}

	log.Infof("delete resource resp:%v", string(buf))

	return c.responseError(buf)
}

// GetResourceRoles 获取资源对应角色.
func (c Client) GetResourceRoles(resID int64) ([]meta.RoleResource, error) {
	url := fmt.Sprintf("http://%s/rbac/role/resource/?resource_id=%v&api=1", c.host, resID)
	buf, err := c.get(url)
	if err != nil {
		log.Infof("Get:%v error:%v", url, err)
		return nil, errors.Trace(err)
	}

	log.Infof("get role resp:%v", string(buf))
	rr := []meta.RoleResource{}
	if err = json.Unmarshal(buf, &rr); err != nil {
		log.Errorf("Get:%v Unmarshal error:%v, buf:%s", url, err, buf)
		return nil, errors.Trace(err)
	}

	return rr, nil
}

// GetResource 获取资源.
func (c Client) GetResource(resID int64) (meta.Resource, error) {
	r := struct {
		Status  int
		Message string
		Data    []meta.Resource
	}{}

	url := fmt.Sprintf("http://%s/rbac/resource/?id=%v", c.host, resID)
	buf, err := c.get(url)
	if err != nil {
		log.Infof("Get:%v error:%v", url, err)
		return meta.Resource{}, errors.Trace(err)
	}

	log.Infof("get resource response:%v", string(buf))

	if err = json.Unmarshal(buf, &r); err != nil {
		log.Errorf("Get:%v Unmarshal error:%v, buf:%s", url, err, buf)
		return meta.Resource{}, errors.Trace(err)
	}

	if r.Status != 0 {
		log.Errorf("Get:%v error:%s", url, r.Message)
		return meta.Resource{}, fmt.Errorf(r.Message)
	}

	return r.Data[0], nil
}

//GetResourceRolesUnrelated 获取未资源对应的所有角色列表.
func (c Client) GetResourceRolesUnrelated(resID int64, email string) ([]meta.Role, error) {
	url := fmt.Sprintf("http://%s/rbac/resource/role/unrelated/?resource_id=%d&email=%s", c.host, resID, email)
	buf, err := c.get(url)
	if err != nil {
		log.Infof("Get:%v error:%v", url, err)
		return nil, errors.Trace(err)
	}

	log.Infof("get role resp:%s", buf)

	r := struct {
		Status  int
		Message string
		Data    []meta.Role
	}{}

	if err = json.Unmarshal(buf, &r); err != nil {
		log.Errorf("Get:%v Unmarshal error:%v, buf:%s", url, err, buf)
		return nil, errors.Trace(err)
	}

	if r.Status != 0 {
		return nil, errors.Errorf(r.Message)
	}

	return r.Data, nil
}

// GetRole 获取角色信息.
func (c Client) GetRole(roleID int64) (meta.Role, error) {
	url := fmt.Sprintf("http://%s/rbac/role/?role_id=%d", c.host, roleID)
	buf, err := c.get(url)
	if err != nil {
		log.Infof("Get:%v error:%v", url, err)
		return meta.Role{}, errors.Trace(err)
	}

	log.Infof("get role resp:%s", buf)

	resp := struct {
		Status  int
		Message string
		Data    []meta.Role
	}{}

	if err = json.Unmarshal(buf, &resp); err != nil {
		log.Errorf("Get:%v Unmarshal error:%v, buf:%s", url, err, buf)
		return meta.Role{}, errors.Trace(err)
	}

	if resp.Status != 0 {
		return meta.Role{}, fmt.Errorf(resp.Message)
	}

	if len(resp.Data) == 0 {
		return meta.Role{}, fmt.Errorf("role:%v not found", roleID)
	}

	return resp.Data[0], nil
}

//GetRoleUsers 根据角色ID邮件获取相关用户.
func (c Client) GetRoleUsers(roleID int64) ([]meta.RoleUser, error) {
	url := fmt.Sprintf("http://%s/rbac/role/user/?role_id=%d", c.host, roleID)
	buf, err := c.get(url)
	if err != nil {
		log.Infof("Get:%v error:%v", url, err)
		return nil, errors.Trace(err)
	}

	log.Infof("get roleUser resp:%v", string(buf))

	resp := struct {
		Status  int
		Message string
		Data    []meta.RoleUser
	}{}

	if err = json.Unmarshal(buf, &resp); err != nil {
		log.Errorf("Get:%v Unmarshal error:%v, buf:%s", url, err, buf)
		return nil, errors.Trace(err)
	}

	if resp.Status != 0 {
		return nil, fmt.Errorf(resp.Message)
	}

	return resp.Data, nil
}

//GetUserRoles 根据邮件获取关联角色信息.
func (c Client) GetUserRoles(email string) ([]meta.RoleUser, error) {
	url := fmt.Sprintf("http://%s/rbac/user/role/?email=%s", c.host, email)
	buf, err := c.get(url)
	if err != nil {
		log.Infof("Get:%v error:%v", url, err)
		return nil, errors.Trace(err)
	}

	log.Infof("get roleUser resp:%v", string(buf))

	resp := struct {
		Status  int
		Message string
		Data    []meta.RoleUser
	}{}

	if err = json.Unmarshal(buf, &resp); err != nil {
		log.Errorf("Get:%v Unmarshal error:%v, buf:%s", url, err, buf)
		return nil, errors.Trace(err)
	}

	if resp.Status != 0 {
		return nil, fmt.Errorf(resp.Message)
	}

	return resp.Data, nil
}

//PostRoleUser 给角色添加用户.
func (c Client) PostRoleUser(roleID int64, user, email string) (int64, error) {
	url := fmt.Sprintf("http://%s/rbac/role/user/?role_id=%d&name=%s&email=%s", c.host, roleID, user, email)
	buf, err := c.post(url, nil)
	if err != nil {
		log.Infof("Put:%v error:%v", url, err)
		return 0, errors.Trace(err)
	}

	log.Infof("add role user resp:%v", string(buf))

	return c.responseID(buf)
}

//PostUser 添加用户.
func (c Client) PostUser(user, email string) (int64, error) {
	url := fmt.Sprintf("http://%s/rbac/user/?name=%s&email=%s", c.host, user, email)
	buf, err := c.post(url, nil)
	if err != nil {
		log.Infof("Post:%v error:%v", url, err)
		return 0, errors.Trace(err)
	}

	log.Infof("add user resp:%v", string(buf))

	return c.responseID(buf)
}

// PutUser 更新用户信息.
func (c Client) PutUser(userID int64, user, email string) error {
	url := fmt.Sprintf("http://%s/rbac/user/?user_id=%v&name=%s&email=%s", c.host, userID, user, email)
	buf, err := c.put(url, nil)
	if err != nil {
		log.Infof("Put:%v error:%v", url, err)
		return errors.Trace(err)
	}

	log.Infof("modify user resp:%s", buf)

	return c.responseError(buf)
}

// DeleteUser 删除用户.
func (c Client) DeleteUser(userID int64) error {
	url := fmt.Sprintf("http://%s/rbac/user/?user_id=%v", c.host, userID)
	buf, err := c.delete(url)
	if err != nil {
		log.Infof("Delete:%v error:%v", url, err)
		return errors.Trace(err)
	}

	log.Infof("delete user resp:%s", buf)

	return c.responseError(buf)
}
