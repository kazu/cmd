CONTAINER_ID=$(shell cat /etc/hostname)
HOST_PWD=$(shell docker inspect $(CONTAINER_ID) -f "{{range  .Mounts}}{{if eq .Destination \"/workspaces\"  }}{{.Source}}/cmd{{end}}{{end}}")


echo:
	echo ${HOST_PWD}