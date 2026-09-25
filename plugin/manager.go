package plugin

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"plugin"
	"strconv"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gotify/server/v3/auth"
	"github.com/gotify/server/v3/model"
	"github.com/gotify/server/v3/plugin/compat"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
)

// The Database interface for encapsulating database access.
type Database interface {
	GetUsers() ([]*model.User, error)
	GetPluginConfByUserAndPath(userid uint, path string) (*model.PluginConf, error)
	CreatePluginConf(p *model.PluginConf) error
	GetPluginConfByApplicationID(appid uint) (*model.PluginConf, error)
	UpdatePluginConf(p *model.PluginConf) error
	CreateMessage(message *model.Message) error
	GetPluginConfByID(id uint) (*model.PluginConf, error)
	GetPluginConfByToken(token string) (*model.PluginConf, error)
	GetUserByID(id uint) (*model.User, error)
	CreateApplication(application *model.Application) error
	UpdateApplication(app *model.Application) error
	GetApplicationsByUser(userID uint) ([]*model.Application, error)
	GetApplicationByToken(token string) (*model.Application, error)
}

// Notifier notifies when a new message was created.
type Notifier interface {
	Notify(userID uint, message *model.MessageExternal)
}

type Dispatcher interface {
	StoreAndDeliver(message *model.Message) (*model.MessageExternal, error)
}

// Manager is an encapsulating layer for plugins and manages all plugins and its instances.
type Manager struct {
	mutex     *sync.RWMutex
	instances map[uint]compat.PluginInstance
	plugins     map[string]compat.Plugin
	pluginFiles map[string]string
	messages    chan MessageWithUserID
	db        Database
	mux       *gin.RouterGroup
	directory  string
	dispatcher Dispatcher
}

// NewManager created a Manager from configurations.
func NewManager(db Database, directory string, mux *gin.RouterGroup, notifier Notifier) (*Manager, error) {
	manager := &Manager{
		mutex:     &sync.RWMutex{},
		instances: map[uint]compat.PluginInstance{},
		plugins:     map[string]compat.Plugin{},
		pluginFiles: map[string]string{},
		messages:    make(chan MessageWithUserID),
		db:        db,
		mux:       mux,
		directory: directory,
	}

	go func() {
		for {
			message := <-manager.messages
			internalMsg := &model.Message{
				ApplicationID: message.Message.ApplicationID,
				Title:         message.Message.Title,
				Priority:      *message.Message.Priority,
				Date:          message.Message.Date,
				Message:       message.Message.Message,
			}
			if message.Message.Extras != nil {
				internalMsg.Extras, _ = json.Marshal(message.Message.Extras)
			}

			manager.mutex.RLock()
			dispatcher := manager.dispatcher
			manager.mutex.RUnlock()
			if dispatcher != nil {
				external, err := dispatcher.StoreAndDeliver(internalMsg)
				if err != nil {
					log.Error().Err(err).Uint("user_id", message.UserID).Msg("Plugin notification delivery failed")
					continue
				}
				if external != nil {
					message.Message.ID = external.ID
				}
				continue
			}

			if err := db.CreateMessage(internalMsg); err != nil {
				log.Error().Err(err).Uint("user_id", message.UserID).Msg("Plugin notification storage failed")
				continue
			}
			message.Message.ID = internalMsg.ID
			notifier.Notify(message.UserID, &message.Message)
		}
	}()

	if err := manager.loadPlugins(directory); err != nil {
		return nil, err
	}

	users, err := manager.db.GetUsers()
	if err != nil {
		return nil, err
	}
	for _, user := range users {
		if err := manager.initializeForUser(*user); err != nil {
			return nil, err
		}
	}

	return manager, nil
}

// MaxPluginUploadBytes is the maximum size accepted for a plugin uploaded through the API.
const MaxPluginUploadBytes int64 = 100 << 20

// ErrAlreadyEnabledOrDisabled is returned on SetPluginEnabled call when a plugin is already enabled or disabled.
var ErrAlreadyEnabledOrDisabled = errors.New("config is already enabled/disabled")

// InstallPlugin persists and loads a Linux Go plugin without requiring a server restart.
// Plugin binaries are server-wide; a disabled per-user plugin configuration is created
// for every existing user just like plugins discovered during normal startup.
func (m *Manager) InstallPlugin(filename string, source io.Reader) (compat.Info, []string, error) {
	var empty compat.Info

	if m.directory == "" {
		return empty, nil, errors.New("plugin installation is disabled because no plugin directory is configured")
	}

	name := filepath.Base(strings.TrimSpace(filename))
	if name == "" || name == "." || strings.HasPrefix(name, ".") {
		return empty, nil, errors.New("invalid plugin filename")
	}
	if strings.ToLower(filepath.Ext(name)) != ".so" {
		return empty, nil, errors.New("plugin file must use the .so extension")
	}

	if err := os.MkdirAll(m.directory, 0o755); err != nil {
		return empty, nil, fmt.Errorf("create plugin directory: %w", err)
	}

	finalPath := filepath.Join(m.directory, name)
	if _, err := os.Stat(finalPath); err == nil {
		return empty, nil, fmt.Errorf("a plugin file named %s already exists", name)
	} else if !os.IsNotExist(err) {
		return empty, nil, fmt.Errorf("check plugin destination: %w", err)
	}

	tmp, err := os.CreateTemp(m.directory, ".gotify-mu-plugin-*.so")
	if err != nil {
		return empty, nil, fmt.Errorf("create temporary plugin file: %w", err)
	}
	tmpPath := tmp.Name()
	cleanupPath := tmpPath
	defer func() {
		if cleanupPath != "" {
			_ = os.Remove(cleanupPath)
		}
	}()

	written, copyErr := io.Copy(tmp, io.LimitReader(source, MaxPluginUploadBytes+1))
	closeErr := tmp.Close()
	if copyErr != nil {
		return empty, nil, fmt.Errorf("write plugin: %w", copyErr)
	}
	if closeErr != nil {
		return empty, nil, fmt.Errorf("close plugin: %w", closeErr)
	}
	if written > MaxPluginUploadBytes {
		return empty, nil, fmt.Errorf("plugin exceeds the %d MiB upload limit", MaxPluginUploadBytes>>20)
	}
	if err := os.Chmod(tmpPath, 0o644); err != nil {
		return empty, nil, fmt.Errorf("set plugin permissions: %w", err)
	}
	if err := os.Rename(tmpPath, finalPath); err != nil {
		return empty, nil, fmt.Errorf("store plugin: %w", err)
	}
	cleanupPath = finalPath

	raw, err := plugin.Open(finalPath)
	if err != nil {
		return empty, nil, pluginFileLoadError{name, err}
	}
	compatPlugin, err := compat.Wrap(raw)
	if err != nil {
		return empty, nil, pluginFileLoadError{name, err}
	}
	info := compatPlugin.PluginInfo()
	if strings.TrimSpace(info.ModulePath) == "" {
		return empty, nil, errors.New("plugin did not report a module path")
	}

	users, err := m.db.GetUsers()
	if err != nil {
		return empty, nil, fmt.Errorf("load users for plugin initialization: %w", err)
	}

	m.mutex.Lock()
	if _, exists := m.plugins[info.ModulePath]; exists {
		m.mutex.Unlock()
		return empty, nil, fmt.Errorf("plugin with module path %s is already installed", info.ModulePath)
	}
	m.plugins[info.ModulePath] = compatPlugin
	m.pluginFiles[info.ModulePath] = finalPath

	warnings := make([]string, 0)
	for _, user := range users {
		userCtx := compat.UserContext{
			ID:    user.ID,
			Name:  user.Name,
			Admin: user.Admin,
		}
		if err := m.initializeSingleUserPlugin(userCtx, compatPlugin); err != nil {
			warnings = append(warnings, fmt.Sprintf("user %d: %s", user.ID, err))
			log.Error().
				Err(err).
				Uint("user_id", user.ID).
				Str("module_path", info.ModulePath).
				Msg("Installed plugin but failed to initialize it for user")
		}
	}
	m.mutex.Unlock()

	cleanupPath = ""
	log.Info().
		Str("path", finalPath).
		Str("module_path", info.ModulePath).
		Msg("Installed plugin from Web UI")

	return info, warnings, nil
}

// SetDispatcher routes plugin notifications through Gotify MU's shared delivery policy engine.
func (m *Manager) SetDispatcher(dispatcher Dispatcher) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.dispatcher = dispatcher
}

// InstallVerifiedPlugin verifies a plugin before loading it into the server.
func (m *Manager) InstallVerifiedPlugin(filename string, source io.Reader, verification InstallVerification) (compat.Info, []string, string, error) {
	var empty compat.Info
	verified, err := verifyPluginStream(m.directory, filename, source, verification)
	if err != nil { return empty, nil, "", err }
	defer os.Remove(verified.Path)

	file, err := os.Open(verified.Path)
	if err != nil { return empty, nil, "", err }
	defer file.Close()

	stem := strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename))
	safeName := fmt.Sprintf("%s-%s.so", stem, verified.SHA256[:12])
	info, warnings, err := m.InstallPlugin(safeName, file)
	return info, warnings, verified.SHA256, err
}

// UninstallPlugin disables a server-wide plugin and removes its binary.
// Existing per-user configuration and historical messages remain so reinstalling
// the same module can recover its prior configuration.
func (m *Manager) UninstallPlugin(modulePath string) error {
	modulePath = strings.TrimSpace(modulePath)
	if modulePath == "" { return errors.New("plugin module path is required") }

	users, err := m.db.GetUsers()
	if err != nil { return err }
	type confEntry struct { conf *model.PluginConf; instance compat.PluginInstance }
	var entries []confEntry
	for _, user := range users {
		conf, confErr := m.db.GetPluginConfByUserAndPath(user.ID, modulePath)
		if confErr != nil { return confErr }
		if conf == nil { continue }
		instance, _ := m.Instance(conf.ID)
		entries = append(entries, confEntry{conf:conf,instance:instance})
	}
	for _, entry := range entries {
		if entry.instance != nil && entry.conf.Enabled {
			if err := entry.instance.Disable(); err != nil {
				return fmt.Errorf("disable plugin for user %d: %w", entry.conf.UserID, err)
			}
		}
		entry.conf.Enabled = false
		if err := m.db.UpdatePluginConf(entry.conf); err != nil { return err }
	}

	m.mutex.Lock()
	for _, entry := range entries { delete(m.instances, entry.conf.ID) }
	path := m.pluginFiles[modulePath]
	delete(m.plugins, modulePath)
	delete(m.pluginFiles, modulePath)
	m.mutex.Unlock()

	if path != "" {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) { return err }
	}
	return nil
}

// StagePluginUpdate verifies and stores a replacement plugin binary. Native Go
// plugins cannot be safely unloaded/reloaded in place, so the replacement takes
// effect on the next Gotify MU restart.
func (m *Manager) StagePluginUpdate(modulePath, filename string, source io.Reader, verification InstallVerification) (string, error) {
	modulePath = strings.TrimSpace(modulePath)
	if modulePath == "" { return "", errors.New("plugin module path is required") }
	m.mutex.RLock()
	currentPath, exists := m.pluginFiles[modulePath]
	m.mutex.RUnlock()
	if !exists || currentPath == "" { return "", errors.New("installed plugin binary not found") }

	verified, err := verifyPluginStream(m.directory, filename, source, verification)
	if err != nil { return "", err }
	defer os.Remove(verified.Path)

	stem := strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename))
	nextPath := filepath.Join(m.directory, fmt.Sprintf("%s-%s.so", stem, verified.SHA256[:12]))
	if err := os.Rename(verified.Path, nextPath); err != nil { return "", err }
	if currentPath != nextPath {
		if err := os.Remove(currentPath); err != nil && !os.IsNotExist(err) {
			_ = os.Remove(nextPath)
			return "", err
		}
	}
	m.mutex.Lock()
	m.pluginFiles[modulePath] = nextPath
	m.mutex.Unlock()
	return verified.SHA256, nil
}

func (m *Manager) HasPlugin(modulePath string) bool {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	_, ok := m.plugins[modulePath]
	return ok
}

// SetPluginEnabled sets the plugins enabled state.
func (m *Manager) SetPluginEnabled(pluginID uint, enabled bool) error {
	instance, err := m.Instance(pluginID)
	if err != nil {
		return errors.New("instance not found")
	}
	conf, err := m.db.GetPluginConfByID(pluginID)
	if err != nil {
		return err
	}

	if conf.Enabled == enabled {
		return ErrAlreadyEnabledOrDisabled
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	if enabled {
		err = instance.Enable()
	} else {
		err = instance.Disable()
	}
	if err != nil {
		return err
	}

	if newConf, err := m.db.GetPluginConfByID(pluginID); /* conf might be updated by instance */ err == nil {
		conf = newConf
	}
	conf.Enabled = enabled
	return m.db.UpdatePluginConf(conf)
}

// PluginInfo returns plugin info.
func (m *Manager) PluginInfo(modulePath string) compat.Info {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if p, ok := m.plugins[modulePath]; ok {
		return p.PluginInfo()
	}
	log.Warn().Str("module_path", modulePath).Msg("Could not get plugin info")
	return compat.Info{
		Name:        "UNKNOWN",
		ModulePath:  modulePath,
		Description: "Oops something went wrong",
	}
}

// Instance returns an instance with the given ID.
func (m *Manager) Instance(pluginID uint) (compat.PluginInstance, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if instance, ok := m.instances[pluginID]; ok {
		return instance, nil
	}
	return nil, errors.New("instance not found")
}

// HasInstance returns whether the given plugin ID has a corresponding instance.
func (m *Manager) HasInstance(pluginID uint) bool {
	instance, err := m.Instance(pluginID)
	return err == nil && instance != nil
}

// RemoveUser disabled all plugins of a user when the user is disabled.
func (m *Manager) RemoveUser(userID uint) error {
	for _, p := range m.plugins {
		pluginConf, err := m.db.GetPluginConfByUserAndPath(userID, p.PluginInfo().ModulePath)
		if err != nil {
			return err
		}
		if pluginConf == nil {
			continue
		}
		if pluginConf.Enabled {
			inst, err := m.Instance(pluginConf.ID)
			if err != nil {
				continue
			}
			m.mutex.Lock()
			err = inst.Disable()
			m.mutex.Unlock()
			if err != nil {
				return err
			}
		}
		m.mutex.Lock()
		delete(m.instances, pluginConf.ID)
		m.mutex.Unlock()
	}
	return nil
}

type pluginFileLoadError struct {
	Filename        string
	UnderlyingError error
}

func (c pluginFileLoadError) Error() string {
	return fmt.Sprintf("error while loading plugin %s: %s", c.Filename, c.UnderlyingError)
}

func (m *Manager) loadPlugins(directory string) error {
	if directory == "" {
		return nil
	}

	pluginFiles, err := os.ReadDir(directory)
	if err != nil {
		return fmt.Errorf("error while reading directory %s", err)
	}
	for _, f := range pluginFiles {
		if f.IsDir() {
			continue
		}

		name := f.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}

		pluginPath := filepath.Join(directory, "./", name)

		log.Info().Str("path", pluginPath).Msg("Loading plugin")
		pRaw, err := plugin.Open(pluginPath)
		if err != nil {
			return pluginFileLoadError{name, err}
		}
		compatPlugin, err := compat.Wrap(pRaw)
		if err != nil {
			return pluginFileLoadError{name, err}
		}
		if err := m.LoadPlugin(compatPlugin); err != nil {
			return pluginFileLoadError{name, err}
		}
		m.pluginFiles[compatPlugin.PluginInfo().ModulePath] = pluginPath
	}
	return nil
}

// LoadPlugin loads a compat plugin, exported to sideload plugins for testing purposes.
func (m *Manager) LoadPlugin(compatPlugin compat.Plugin) error {
	modulePath := compatPlugin.PluginInfo().ModulePath
	if _, ok := m.plugins[modulePath]; ok {
		return fmt.Errorf("plugin with module path %s is present at least twice", modulePath)
	}
	m.plugins[modulePath] = compatPlugin
	return nil
}

// InitializeForUserID initializes all plugin instances for a given user.
func (m *Manager) InitializeForUserID(userID uint) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	user, err := m.db.GetUserByID(userID)
	if err != nil {
		return err
	}
	if user != nil {
		return m.initializeForUser(*user)
	}
	return fmt.Errorf("user with id %d not found", userID)
}

func (m *Manager) initializeForUser(user model.User) error {
	userCtx := compat.UserContext{
		ID:    user.ID,
		Name:  user.Name,
		Admin: user.Admin,
	}

	for _, p := range m.plugins {
		if err := m.initializeSingleUserPlugin(userCtx, p); err != nil {
			return err
		}
	}

	apps, err := m.db.GetApplicationsByUser(user.ID)
	if err != nil {
		return err
	}
	for _, app := range apps {
		conf, err := m.db.GetPluginConfByApplicationID(app.ID)
		if err != nil {
			return err
		}
		if conf != nil {
			_, compatExist := m.plugins[conf.ModulePath]
			app.Internal = compatExist
		} else {
			app.Internal = false
		}
		m.db.UpdateApplication(app)
	}

	return nil
}

func (m *Manager) initializeSingleUserPlugin(userCtx compat.UserContext, p compat.Plugin) error {
	info := p.PluginInfo()
	instance := p.NewPluginInstance(userCtx)
	userID := userCtx.ID

	pluginConf, err := m.db.GetPluginConfByUserAndPath(userID, info.ModulePath)
	if err != nil {
		return err
	}

	if pluginConf == nil {
		var err error
		pluginConf, err = m.createPluginConf(instance, info, userID)
		if err != nil {
			return err
		}
	}

	m.instances[pluginConf.ID] = instance

	if compat.HasSupport(instance, compat.Messenger) {
		if pluginConf.ApplicationID == 0 {
			// The Messenger capability was added after this plugin was first
			// initialized for the user, so no internal application exists yet.
			// Create one now, otherwise messages would be stored with
			// application_id = 0 and become orphaned (not shown, not deletable).
			app, err := m.createInternalApplication(info, userID)
			if err != nil {
				return err
			}
			pluginConf.ApplicationID = app.ID
			if err := m.db.UpdatePluginConf(pluginConf); err != nil {
				return err
			}
		}
		instance.SetMessageHandler(redirectToChannel{
			ApplicationID: pluginConf.ApplicationID,
			UserID:        pluginConf.UserID,
			Messages:      m.messages,
		})
	}
	if compat.HasSupport(instance, compat.Storager) {
		instance.SetStorageHandler(dbStorageHandler{pluginConf.ID, m.db})
	}
	if compat.HasSupport(instance, compat.Configurer) {
		m.initializeConfigurerForSingleUserPlugin(instance, pluginConf)
	}
	if compat.HasSupport(instance, compat.Webhooker) {
		id := pluginConf.ID
		g := m.mux.Group(pluginConf.Token+"/", requirePluginEnabled(id, m.db))
		instance.RegisterWebhook(strings.Replace(g.BasePath(), ":id", strconv.Itoa(int(id)), 1), g)
	}
	if pluginConf.Enabled {
		err := instance.Enable()
		if err != nil {
			// Single user plugin cannot be enabled
			// Don't panic, disable for now and wait for user to update config
			log.Warn().Err(err).Str("user", userCtx.Name).Msg("Plugin initialize failed, disabling now")
			pluginConf.Enabled = false
			m.db.UpdatePluginConf(pluginConf)
		}
	}
	return nil
}

func (m *Manager) initializeConfigurerForSingleUserPlugin(instance compat.PluginInstance, pluginConf *model.PluginConf) {
	if len(pluginConf.Config) == 0 {
		// The Configurer is newly implemented
		// Use the default config
		pluginConf.Config, _ = yaml.Marshal(instance.DefaultConfig())
		m.db.UpdatePluginConf(pluginConf)
	}
	c := instance.DefaultConfig()
	if yaml.Unmarshal(pluginConf.Config, c) != nil || instance.ValidateAndSetConfig(c) != nil {
		pluginConf.Enabled = false

		log.Warn().
			Str("module_path", pluginConf.ModulePath).
			Uint("user_id", pluginConf.UserID).
			Msg("Plugin failed to initialize because it rejected the current config. It might be outdated. A default config is used and the user would need to enable it again.")
		newConf := bytes.NewBufferString("# Plugin initialization failed because it rejected the current config. It might be outdated.\r\n# A default plugin configuration is used:\r\n")

		d, _ := yaml.Marshal(c)
		newConf.Write(d)
		newConf.WriteString("\r\n")

		newConf.WriteString("# The original configuration: \r\n")
		oldConf := bufio.NewScanner(bytes.NewReader(pluginConf.Config))
		for oldConf.Scan() {
			newConf.WriteString("# ")
			newConf.WriteString(oldConf.Text())
			newConf.WriteString("\r\n")
		}

		pluginConf.Config = newConf.Bytes()

		m.db.UpdatePluginConf(pluginConf)
		instance.ValidateAndSetConfig(instance.DefaultConfig())
	}
}

func (m *Manager) createPluginConf(instance compat.PluginInstance, info compat.Info, userID uint) (*model.PluginConf, error) {
	pluginConf := &model.PluginConf{
		UserID:     userID,
		ModulePath: info.ModulePath,
		Token:      auth.GeneratePluginToken(),
	}
	if compat.HasSupport(instance, compat.Configurer) {
		pluginConf.Config, _ = yaml.Marshal(instance.DefaultConfig())
	}
	if compat.HasSupport(instance, compat.Messenger) {
		app, err := m.createInternalApplication(info, userID)
		if err != nil {
			return nil, err
		}
		pluginConf.ApplicationID = app.ID
	}
	if err := m.db.CreatePluginConf(pluginConf); err != nil {
		return nil, err
	}
	return pluginConf, nil
}

// createInternalApplication creates the auto generated internal application a
// Messenger plugin uses to publish its messages.
func (m *Manager) createInternalApplication(info compat.Info, userID uint) (*model.Application, error) {
	tokenPublic, _ := auth.GenerateApplicationToken()
	app := &model.Application{
		Token:       tokenPublic,
		Name:        info.String(),
		UserID:      userID,
		Internal:    true,
		Description: fmt.Sprintf("auto generated application for %s", info.ModulePath),
	}
	if err := m.db.CreateApplication(app); err != nil {
		return nil, err
	}
	return app, nil
}
