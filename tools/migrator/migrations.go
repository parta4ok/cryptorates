package migrator

import (
	"log/slog"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
	"github.com/pkg/errors"
)

var (
	ErrInvalidParam = errors.New("invalid param")
	ErrInternal     = errors.New("internal error")
)

type MigrationTask struct {
	ServiceName      string `koanf:"service_name"`
	ConnectionString string `koanf:"connection_string"`
	MigrationPath    string `koanf:"migration_path"`
}

type Config struct {
	DBType     string                     `koanf:"db_type"`
	Migrations map[string][]MigrationTask `koanf:"migrations"`
}

type TaskKeeper struct {
	parser *koanf.Koanf
}

func NewTaskKeeper(configPath string) (*TaskKeeper, error) {
	if configPath == "" {
		return nil, errors.WithMessage(ErrInvalidParam, "config path is empty")
	}

	parser := koanf.New(".")
	if err := parser.Load(file.Provider(configPath), yaml.Parser()); err != nil {
		return nil, errors.WithMessagef(ErrInternal, "failed to load config: %v", err)
	}

	return &TaskKeeper{
		parser: parser,
	}, nil
}

func (tk *TaskKeeper) ParseMigrationTasks(dbType string) ([]MigrationTask, error) {
	var config Config
	if err := tk.parser.Unmarshal("", &config); err != nil {
		return nil, errors.WithMessagef(ErrInternal, "failed to unmarshal config: %v", err)
	}

	slog.Info("config", slog.Any("migrations", config.Migrations))

	return config.Migrations[dbType], nil
}
