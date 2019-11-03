# ROOT_PACKAGE :: the package (relative to $GOPATH/src) that is the target for code generation
#ROOT_PACKAGE="github.com/trstringer/k8s-controller-core-resource"
ROOT_PACKAGE="github.com/llimon/argo-scheduler"
# CUSTOM_RESOURCE_NAME :: the name of the custom resource that we're generating client code for
CUSTOM_RESOURCE_NAME="argoscheduler"
# CUSTOM_RESOURCE_VERSION :: the version of the resource
CUSTOM_RESOURCE_VERSION="v1"

# retrieve the code-generator scripts and bins
go get -u k8s.io/code-generator/...
cd $GOPATH/src/k8s.io/code-generator

# run the code-generator entrypoint script
./generate-groups.sh all "$ROOT_PACKAGE/pkg/client" "$ROOT_PACKAGE/pkg/apis" "$CUSTOM_RESOURCE_NAME:$CUSTOM_RESOURCE_VERSION"

# view the newly generated files
tree $GOPATH/src/$ROOT_PACKAGE/pkg/client
# pkg/client/
# ├── clientset
# │   └── versioned
# │       ├── clientset.go
# │       ├── doc.go
# │       ├── fake
# │       │   ├── clientset_generated.go
# │       │   ├── doc.go
# │       │   └── register.go
# │       ├── scheme
# │       │   ├── doc.go
# │       │   └── register.go
# │       └── typed
# │           └── myresource
# │               └── v1
# │                   ├── doc.go
# │                   ├── fake
# │                   │   ├── doc.go
# │                   │   ├── fake_myresource_client.go
# │                   │   └── fake_myresource.go
# │                   ├── generated_expansion.go
# │                   ├── myresource_client.go
# │                   └── myresource.go
# ├── informers
# │   └── externalversions
# │       ├── factory.go
# │       ├── generic.go
# │       ├── internalinterfaces
# │       │   └── factory_interfaces.go
# │       └── myresource
# │           ├── interface.go
# │           └── v1
# │               ├── interface.go
# │               └── myresource.go
# └── listers
#     └── myresource
#         └── v1
#             ├── expansion_generated.go
#             └── myresource.go
# 
# 16 directories, 22 files
