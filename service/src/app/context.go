package app

import (
	"ivory/src/config"
	. "ivory/src/model"
	"ivory/src/persistence"
	"ivory/src/router"
	"ivory/src/service"
)

type Context struct {
	authRouter     *router.AuthRouter
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

	// DB
	db := config.NewBoltDB("ivory.db")
	clusterBucket := config.NewBoltBucket[Cluster](db, "Cluster")
	compactTableBucket := config.NewBoltBucket[Bloat](db, "CompactTable")
	certBucket := config.NewBoltBucket[Cert](db, "Cert")
	tagBucket := config.NewBoltBucket[[]string](db, "Tag")
	secretBucket := config.NewBoltBucket[string](db, "Secret")
	passwordBucket := config.NewBoltBucket[Password](db, "Password")
	queryBucket := config.NewBoltBucket[Query](db, "Query")

	// FILES
	compactTableFiles := config.NewFileGateway("pgcompacttable", ".log")
	certFiles := config.NewFileGateway("cert", ".crt")

	// REPOS
	clusterRepo := persistence.NewClusterRepository(clusterBucket)
	bloatRepository := persistence.NewBloatRepository(compactTableBucket, compactTableFiles)
	certRepo := persistence.NewCertRepository(certBucket, certFiles)
	tagRepo := persistence.NewTagRepository(tagBucket)
	secretRepo := persistence.NewSecretRepository(secretBucket)
	passwordRepo := persistence.NewPasswordRepository(passwordBucket)
	queryRepo := persistence.NewQueryRepository(queryBucket)

	// SERVICES
	encryptionService := service.NewEncryptionService()
	secretService := service.NewSecretService(secretRepo, encryptionService)
	passwordService := service.NewPasswordService(passwordRepo, secretService, encryptionService)
	certService := service.NewCertService(certRepo)

	postgresGateway := service.NewPostgresClient(passwordService)
	sidecarClient := service.NewSidecarClient(passwordService, certService)
	patroniGateway := service.NewPatroniGateway(sidecarClient)

	tagService := service.NewTagService(tagRepo)
	queryService := service.NewQueryService(queryRepo, secretService, postgresGateway)
	clusterService := service.NewClusterService(clusterRepo, tagService, patroniGateway)
	bloatService := service.NewBloatService(bloatRepository, passwordService)
	authService := service.NewAuthService(secretService)
	eraseService := service.NewEraseService(
		passwordService,
		clusterService,
		certService,
		tagService,
		bloatService,
		queryService,
		secretService,
	)

	// TODO refactor: shouldn't router use repos? consider change to service usage (possible cycle dependencies problems)
	// TODO repos -> services / gateway -> routers, can service use service? can service use repo that belongs to another service?
	return &Context{
		infoRouter:     router.NewInfoRouter(env, secretService, authService),
		authRouter:     router.NewAuthRouter(env, authService),
		clusterRouter:  router.NewClusterRouter(clusterService),
		bloatRouter:    router.NewBloatRouter(bloatService),
		certRouter:     router.NewCertRouter(certService),
		secretRouter:   router.NewSecretRouter(secretService, passwordService),
		passwordRouter: router.NewPasswordRouter(passwordService),
		tagRouter:      router.NewTagRouter(tagService),
		instanceRouter: router.NewInstanceRouter(patroniGateway),
		queryRouter:    router.NewQueryRouter(queryService, postgresGateway),
		eraseRouter:    router.NewEraseRouter(eraseService),
	}
}
