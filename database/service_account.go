package database

import (
	"time"

	"github.com/gotify/server/v3/model"
	"gorm.io/gorm"
)

func (d *GormDatabase) GetServiceAccounts() ([]*model.ServiceAccount,error) {
	var items []*model.ServiceAccount
	return items,d.DB.Order("name asc, id asc").Find(&items).Error
}
func (d *GormDatabase) GetServiceAccountByID(id uint)(*model.ServiceAccount,error){
	item:=new(model.ServiceAccount)
	if err:=d.DB.First(item,id).Error;err!=nil{if err==gorm.ErrRecordNotFound{return nil,nil};return nil,err}
	return item,nil
}
func (d *GormDatabase) SaveServiceAccount(item *model.ServiceAccount)error{return d.DB.Save(item).Error}
func (d *GormDatabase) DeleteServiceAccount(id uint)error{
	return d.DB.Transaction(func(tx *gorm.DB)error{
		if err:=tx.Where("account_id = ?",id).Delete(&model.ServiceAccountToken{}).Error;err!=nil{return err}
		return tx.Delete(&model.ServiceAccount{},id).Error
	})
}
func (d *GormDatabase) GetServiceAccountTokens(accountID uint)([]*model.ServiceAccountToken,error){
	var items []*model.ServiceAccountToken
	return items,d.DB.Where("account_id = ?",accountID).Order("created_at desc").Find(&items).Error
}
func (d *GormDatabase) SaveServiceAccountToken(item *model.ServiceAccountToken)error{return d.DB.Save(item).Error}
func (d *GormDatabase) DeleteServiceAccountToken(accountID,id uint)error{
	return d.DB.Where("account_id = ? AND id = ?",accountID,id).Delete(&model.ServiceAccountToken{}).Error
}
func (d *GormDatabase) GetServiceAccountByTokenHash(hash string,now time.Time)(*model.ServiceAccount,*model.ServiceAccountToken,error){
	token:=new(model.ServiceAccountToken)
	if err:=d.DB.Where("token_hash = ? AND (expires_at IS NULL OR expires_at > ?)",hash,now).First(token).Error;err!=nil{
		if err==gorm.ErrRecordNotFound{return nil,nil,nil};return nil,nil,err
	}
	account:=new(model.ServiceAccount)
	if err:=d.DB.First(account,token.AccountID).Error;err!=nil{
		if err==gorm.ErrRecordNotFound{return nil,nil,nil};return nil,nil,err
	}
	if !account.Enabled{return nil,nil,nil}
	return account,token,nil
}
func (d *GormDatabase) TouchServiceAccountToken(accountID,tokenID uint,now time.Time)error{
	if err:=d.DB.Model(&model.ServiceAccountToken{}).Where("id = ? AND account_id = ?",tokenID,accountID).Update("last_used_at",&now).Error;err!=nil{return err}
	return d.DB.Model(&model.ServiceAccount{}).Where("id = ?",accountID).Update("last_used_at",&now).Error
}
