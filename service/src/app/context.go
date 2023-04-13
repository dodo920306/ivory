package app

import (
	"ivory/src/config"
	. "ivory/src/model"
	"ivory/src/persistence"
	"ivory/src/router"
	"ivory/src/service"
)

type Context struct {
	env            *config.Env
	infoRouter     *router.InfoRouter
	clusterRouter  *router.ClusterRouter
	bloatRouter    *router.BloatRouter
	certRouter     *router.CertRouter
	secretRouter   *router.SecretRouter
	passwordRouter *router.PasswordRouter
	tagRouter      *router.TagRouter
	instanceRouter *router.InstanceRouter
	queryRouter    *router.QueryRouter
	eraseRouter    *router.EraseRouter
}

func NewContext() *Context {
	env := config.NewEnv()

	db := config.NewBoltDB("ivory.db")
	clusterBucket := config.NewBoltBucket[ClusterModel](db, "Cluster")
	compactTableBucket := config.NewBoltBucket[CompactTableModel](db, "CompactTable")
	certBucket := config.NewBoltBucket[CertModel](db, "Cert")
	tagBucket := config.NewBoltBucket[[]string](db, "Tag")
	secretBucket := config.NewBoltBucket[string](db, "Secret")
	passwordBucket := config.NewBoltBucket[Credential](db, "Password")
	queryBucket := config.NewBoltBucket[Query](db, "Query")

	compactTableFiles := config.NewFileGateway("pgcompacttable", ".log")
	certFiles := config.NewFileGateway("cert", ".crt")

	clusterRepo := persistence.NewClusterRepository(clusterBucket)
	compactTableRepo := persistence.NewCompactTableRepository(compactTableBucket, compactTableFiles)
	certRepo := persistence.NewCertRepository(certBucket, certFiles)
	tagRepo := persistence.NewTagRepository(tagBucket)
	secretRepo := persistence.NewSecretRepository(secretBucket)
	passwordRepo := persistence.NewPasswordRepository(passwordBucket)
	queryRepo := persistence.NewQueryRepository(queryBucket)

	encryption := service.NewEncryption()
	secretService := service.NewSecretService(secretRepo, encryption)
	passwordService := service.NewPasswordService(passwordRepo, secretService, encryption)
	clusterService := service.NewClusterService(clusterRepo)
	sidecarGateway := service.NewSidecarGateway(clusterService, certRepo, passwordService)
	postgresGateway := service.NewPostgresGateway(clusterService, passwordService)
	queryService := service.NewQueryService(queryRepo, postgresGateway, secretService)
	bloatService := service.NewBloatService(compactTableRepo, passwordRepo, compactTableFiles, secretService, encryption)
	patroniService := service.NewPatroniService(sidecarGateway)
	eraseService := service.NewEraseService(passwordRepo, clusterService, certRepo, tagRepo, compactTableRepo, queryService, secretService)

	// TODO refactor: shouldn't router use repos? consider change to service usage (possible cycle dependencies problems)
	// TODO repos -> services / gateway -> routers, can service use service? can service use repo that belongs to another service?
	return &Context{
		env:            env,
		infoRouter:     router.NewInfoRouter(env, secretService),
		clusterRouter:  router.NewClusterRouter(clusterService, tagRepo),
		bloatRouter:    router.NewBloatRouter(bloatService, compactTableRepo),
		certRouter:     router.NewCertRouter(certRepo),
		secretRouter:   router.NewSecretRouter(secretService, passwordService),
		passwordRouter: router.NewPasswordRouter(passwordService),
		tagRouter:      router.NewTagRouter(tagRepo),
		instanceRouter: router.NewInstanceRouter(patroniService),
		queryRouter:    router.NewQueryRouter(queryService, postgresGateway),
		eraseRouter:    router.NewEraseRouter(eraseService),
	}
}
