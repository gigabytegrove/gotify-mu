package database

import (
	"strings"
	"time"

	"github.com/gotify/server/v3/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (d *GormDatabase) GetEmailGateways() ([]*model.EmailGateway, error) {
	var items []*model.EmailGateway
	if err := d.DB.Order("name asc, id asc").Find(&items).Error; err != nil { return nil, err }
	for _, item := range items {
		plain, err := d.Secrets.Decrypt(item.Password)
		if err != nil { return nil, err }
		item.Password = plain
	}
	return items, nil
}
func (d *GormDatabase) GetEmailGatewayByID(id uint) (*model.EmailGateway, error) {
	item := new(model.EmailGateway)
	if err := d.DB.First(item, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound { return nil, nil }
		return nil, err
	}
	plain, err := d.Secrets.Decrypt(item.Password)
	if err != nil { return nil, err }
	item.Password = plain
	return item, nil
}
func (d *GormDatabase) SaveEmailGateway(item *model.EmailGateway) error {
	plain, err := d.Secrets.Decrypt(item.Password)
	if err != nil { return err }
	encrypted, err := d.Secrets.Encrypt(plain)
	if err != nil { return err }
	item.Password = encrypted
	err = d.DB.Save(item).Error
	item.Password = plain
	return err
}
func (d *GormDatabase) DeleteEmailGateway(id uint) error { return d.DB.Delete(&model.EmailGateway{}, id).Error }
func (d *GormDatabase) GetEmailGatewaysForMessage(applicationID uint, priority int) ([]*model.EmailGateway, error) {
	var items []*model.EmailGateway
	if err := d.DB.Where("enabled = ? AND source_application_id = ? AND min_priority <= ?", true, applicationID, priority).Find(&items).Error; err != nil {
		return nil, err
	}
	for _, item := range items {
		plain, err := d.Secrets.Decrypt(item.Password)
		if err != nil { return nil, err }
		item.Password = plain
	}
	return items, nil
}
func (d *GormDatabase) UpdateEmailGatewayStatus(id uint, sentAt *time.Time, lastError string, errorAt *time.Time) error {
	status := "ready"
	if lastError != "" { status = "error" }
	updates := map[string]any{"status":status,"last_error":lastError,"last_error_at":errorAt}
	if sentAt != nil { updates["last_sent_at"] = sentAt }
	return d.DB.Model(&model.EmailGateway{}).Where("id = ?", id).Updates(updates).Error
}

func (d *GormDatabase) GetSMTPRoutes() ([]*model.SMTPRoute, error) {
	var items []*model.SMTPRoute
	if err := d.DB.Order("name asc, id asc").Find(&items).Error; err != nil { return nil, err }
	for _, item := range items {
		plain, err := d.Secrets.Decrypt(item.Password)
		if err != nil { return nil, err }
		item.Password = plain
	}
	return items, nil
}
func (d *GormDatabase) GetSMTPRouteByID(id uint) (*model.SMTPRoute, error) {
	item := new(model.SMTPRoute)
	if err := d.DB.First(item, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound { return nil, nil }
		return nil, err
	}
	plain, err := d.Secrets.Decrypt(item.Password)
	if err != nil { return nil, err }
	item.Password = plain
	return item, nil
}
func (d *GormDatabase) SaveSMTPRoute(item *model.SMTPRoute) error {
	plain, err := d.Secrets.Decrypt(item.Password)
	if err != nil { return err }
	encrypted, err := d.Secrets.Encrypt(plain)
	if err != nil { return err }
	item.Password = encrypted
	err = d.DB.Save(item).Error
	item.Password = plain
	return err
}
func (d *GormDatabase) DeleteSMTPRoute(id uint) error { return d.DB.Delete(&model.SMTPRoute{}, id).Error }
func (d *GormDatabase) MatchSMTPRoute(recipient string) (*model.SMTPRoute, error) {
	recipient = strings.ToLower(strings.TrimSpace(recipient))
	var items []*model.SMTPRoute
	if err := d.DB.Where("enabled = ?", true).Order("id asc").Find(&items).Error; err != nil { return nil, err }
	for _, item := range items {
		pattern := strings.ToLower(strings.TrimSpace(item.Recipient))
		if pattern == recipient || (strings.HasPrefix(pattern, "*@") && strings.HasSuffix(recipient, pattern[1:])) {
			plain, err := d.Secrets.Decrypt(item.Password)
			if err != nil { return nil, err }
			item.Password = plain
			return item, nil
		}
	}
	return nil, nil
}

func (d *GormDatabase) GetRSSMonitors() ([]*model.RSSMonitor, error) {
	var items []*model.RSSMonitor
	return items, d.DB.Order("name asc, id asc").Find(&items).Error
}
func (d *GormDatabase) GetRSSMonitorByID(id uint) (*model.RSSMonitor, error) {
	item := new(model.RSSMonitor)
	if err := d.DB.First(item, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound { return nil, nil }
		return nil, err
	}
	return item, nil
}
func (d *GormDatabase) SaveRSSMonitor(item *model.RSSMonitor) error { return d.DB.Save(item).Error }
func (d *GormDatabase) DeleteRSSMonitor(id uint) error {
	return d.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("connector_type = ? AND connector_id = ?", "rss", id).Delete(&model.ConnectorSeenItem{}).Error; err != nil { return err }
		return tx.Delete(&model.RSSMonitor{}, id).Error
	})
}
func (d *GormDatabase) GetDueRSSMonitors(now time.Time) ([]*model.RSSMonitor, error) {
	var items []*model.RSSMonitor
	return items, d.DB.Where(
		"enabled = ? AND (last_checked_at IS NULL OR last_checked_at <= ?)",
		true, now.Add(-time.Minute),
	).Find(&items).Error
}
func (d *GormDatabase) UpdateRSSMonitorStatus(item *model.RSSMonitor) error { return d.DB.Save(item).Error }

func (d *GormDatabase) GetSyslogRoutes() ([]*model.SyslogRoute, error) {
	var items []*model.SyslogRoute
	return items, d.DB.Order("name asc, id asc").Find(&items).Error
}
func (d *GormDatabase) GetSyslogRouteByID(id uint) (*model.SyslogRoute, error) {
	item := new(model.SyslogRoute)
	if err := d.DB.First(item, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound { return nil, nil }
		return nil, err
	}
	return item, nil
}
func (d *GormDatabase) SaveSyslogRoute(item *model.SyslogRoute) error { return d.DB.Save(item).Error }
func (d *GormDatabase) DeleteSyslogRoute(id uint) error { return d.DB.Delete(&model.SyslogRoute{}, id).Error }

func (d *GormDatabase) GetCalendarMonitors() ([]*model.CalendarMonitor, error) {
	var items []*model.CalendarMonitor
	return items, d.DB.Order("name asc, id asc").Find(&items).Error
}
func (d *GormDatabase) GetCalendarMonitorByID(id uint) (*model.CalendarMonitor, error) {
	item := new(model.CalendarMonitor)
	if err := d.DB.First(item, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound { return nil, nil }
		return nil, err
	}
	return item, nil
}
func (d *GormDatabase) SaveCalendarMonitor(item *model.CalendarMonitor) error { return d.DB.Save(item).Error }
func (d *GormDatabase) DeleteCalendarMonitor(id uint) error {
	return d.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("connector_type = ? AND connector_id = ?", "ical", id).Delete(&model.ConnectorSeenItem{}).Error; err != nil { return err }
		return tx.Delete(&model.CalendarMonitor{}, id).Error
	})
}
func (d *GormDatabase) GetDueCalendarMonitors(now time.Time) ([]*model.CalendarMonitor, error) {
	var items []*model.CalendarMonitor
	return items, d.DB.Where(
		"enabled = ? AND (last_checked_at IS NULL OR last_checked_at <= ?)",
		true, now.Add(-time.Minute),
	).Find(&items).Error
}

func (d *GormDatabase) MarkConnectorItemSeen(connectorType string, connectorID uint, itemKey string, now time.Time) (bool, error) {
	item := &model.ConnectorSeenItem{ConnectorType:connectorType,ConnectorID:connectorID,ItemKey:itemKey,SeenAt:now}
	result := d.DB.Clauses(clause.OnConflict{DoNothing:true}).Create(item)
	return result.RowsAffected > 0, result.Error
}

func (d *GormDatabase) MarkConnectorItemSeenWithin(connectorType string, connectorID uint, itemKey string, now time.Time, window time.Duration) (bool, error) {
	if window <= 0 {
		return true, nil
	}
	cutoff := now.Add(-window)
	result := d.DB.Model(&model.ConnectorSeenItem{}).
		Where("connector_type = ? AND connector_id = ? AND item_key = ? AND seen_at <= ?", connectorType, connectorID, itemKey, cutoff).
		Update("seen_at", now)
	if result.Error != nil { return false, result.Error }
	if result.RowsAffected > 0 { return true, nil }
	item := &model.ConnectorSeenItem{ConnectorType:connectorType,ConnectorID:connectorID,ItemKey:itemKey,SeenAt:now}
	created := d.DB.Clauses(clause.OnConflict{DoNothing:true}).Create(item)
	if created.Error != nil { return false, created.Error }
	return created.RowsAffected > 0, nil
}

func (d *GormDatabase) CleanupConnectorSeenItems(before time.Time) error {
	return d.DB.Where("seen_at < ?", before).Delete(&model.ConnectorSeenItem{}).Error
}
