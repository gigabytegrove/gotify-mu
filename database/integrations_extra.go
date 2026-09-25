package database

import (
	"errors"
	"fmt"

	"github.com/gotify/server/v3/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (d *GormDatabase) GetRSSIntegrations() ([]*model.RSSIntegration, error) {
	var items []*model.RSSIntegration
	return items, d.DB.Order("name asc, id asc").Find(&items).Error
}
func (d *GormDatabase) GetRSSIntegrationByID(id uint) (*model.RSSIntegration, error) {
	item := new(model.RSSIntegration)
	if err := d.DB.First(item, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) { return nil, nil }
		return nil, err
	}
	return item, nil
}
func (d *GormDatabase) SaveRSSIntegration(item *model.RSSIntegration) error { return d.DB.Save(item).Error }
func (d *GormDatabase) DeleteRSSIntegration(id uint) error {
	return d.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("kind = ? AND object_id = ?", "rss", id).Delete(&model.IntegrationStatus{}).Error; err != nil { return err }
		if err := tx.Where("key = ?", integrationLeaseKey("rss", id)).Delete(&model.AutomationLease{}).Error; err != nil { return err }
		return tx.Delete(&model.RSSIntegration{}, id).Error
	})
}

func (d *GormDatabase) GetCalendarIntegrations() ([]*model.CalendarIntegration, error) {
	var items []*model.CalendarIntegration
	return items, d.DB.Order("name asc, id asc").Find(&items).Error
}
func (d *GormDatabase) GetCalendarIntegrationByID(id uint) (*model.CalendarIntegration, error) {
	item := new(model.CalendarIntegration)
	if err := d.DB.First(item, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) { return nil, nil }
		return nil, err
	}
	return item, nil
}
func (d *GormDatabase) SaveCalendarIntegration(item *model.CalendarIntegration) error { return d.DB.Save(item).Error }
func (d *GormDatabase) DeleteCalendarIntegration(id uint) error {
	return d.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("kind = ? AND object_id = ?", "calendar", id).Delete(&model.IntegrationStatus{}).Error; err != nil { return err }
		if err := tx.Where("key = ?", integrationLeaseKey("calendar", id)).Delete(&model.AutomationLease{}).Error; err != nil { return err }
		return tx.Delete(&model.CalendarIntegration{}, id).Error
	})
}

func (d *GormDatabase) GetEmailGateways() ([]*model.EmailGateway, error) {
	var items []*model.EmailGateway
	return items, d.DB.Order("name asc, id asc").Find(&items).Error
}
func (d *GormDatabase) GetEmailGatewayByID(id uint) (*model.EmailGateway, error) {
	item := new(model.EmailGateway)
	if err := d.DB.First(item, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) { return nil, nil }
		return nil, err
	}
	return item, nil
}
func (d *GormDatabase) GetEmailGatewaysForMessage(applicationID uint, priority int) ([]*model.EmailGateway, error) {
	var items []*model.EmailGateway
	return items, d.DB.Where("application_id = ? AND enabled = ? AND min_priority <= ?", applicationID, true, priority).
		Order("id asc").Find(&items).Error
}
func (d *GormDatabase) SaveEmailGateway(item *model.EmailGateway) error { return d.DB.Save(item).Error }
func (d *GormDatabase) DeleteEmailGateway(id uint) error {
	return d.DB.Where("id = ?", id).Delete(&model.EmailGateway{}).Error
}

func (d *GormDatabase) GetSMTPReceiver() (*model.SMTPReceiver, error) {
	item := new(model.SMTPReceiver)
	if err := d.DB.First(item, 1).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &model.SMTPReceiver{ID:1, ListenAddress:":2525", MaxMessageBytes:10<<20}, nil
		}
		return nil, err
	}
	return item, nil
}
func (d *GormDatabase) SaveSMTPReceiver(item *model.SMTPReceiver) error {
	item.ID = 1
	return d.DB.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name:"id"}},
		DoUpdates: clause.AssignmentColumns([]string{"listen_address","username","password","allowed_cid_rs","max_message_bytes","enabled","updated_at"}),
	}).Create(item).Error
}
func (d *GormDatabase) GetSMTPRoutes() ([]*model.SMTPRoute, error) {
	var items []*model.SMTPRoute
	return items, d.DB.Order("recipient asc").Find(&items).Error
}
func (d *GormDatabase) GetSMTPRouteByRecipient(recipient string) (*model.SMTPRoute, error) {
	item := new(model.SMTPRoute)
	if err := d.DB.Where("lower(recipient) = lower(?) AND enabled = ?", recipient, true).First(item).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) { return nil, nil }
		return nil, err
	}
	return item, nil
}
func (d *GormDatabase) SaveSMTPRoute(item *model.SMTPRoute) error { return d.DB.Save(item).Error }
func (d *GormDatabase) DeleteSMTPRoute(id uint) error { return d.DB.Delete(&model.SMTPRoute{}, id).Error }

func (d *GormDatabase) GetSyslogReceivers() ([]*model.SyslogReceiver, error) {
	var items []*model.SyslogReceiver
	return items, d.DB.Order("name asc, id asc").Find(&items).Error
}
func (d *GormDatabase) GetSyslogReceiverByID(id uint) (*model.SyslogReceiver, error) {
	item := new(model.SyslogReceiver)
	if err := d.DB.First(item, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) { return nil, nil }
		return nil, err
	}
	return item, nil
}
func (d *GormDatabase) SaveSyslogReceiver(item *model.SyslogReceiver) error { return d.DB.Save(item).Error }
func (d *GormDatabase) DeleteSyslogReceiver(id uint) error {
	return d.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("kind = ? AND object_id = ?", "syslog", id).Delete(&model.IntegrationStatus{}).Error; err != nil { return err }
		if err := tx.Where("key = ?", integrationLeaseKey("syslog", id)).Delete(&model.AutomationLease{}).Error; err != nil { return err }
		return tx.Delete(&model.SyslogReceiver{}, id).Error
	})
}

func integrationLeaseKey(kind string, id uint) string {
	return "integration:" + kind + ":" + fmt.Sprint(id)
}
