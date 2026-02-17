all: linux windows web

linux:
	GOOS=linux GOARCH=amd64 go build -o defective
	chmod +x defective

	mkdir -p defective-linux-amd64
	cp defective defective-linux-amd64/

	tar -czvf defective-linux-amd64.tar.gz defective-linux-amd64
	butler push defective milk9111/defective:linux

windows:
	GOOS=windows GOARCH=amd64 go build -o defective.exe
	butler push defective.exe milk9111/defective:windows

# mac-amd64: # glfw doesn't support it
# 	GOOS=darwin GOARCH=amd64 go build -o defective
# 	butler push defective milk9111/defective:mac-amd64 

# mac-arm64: # glfw doesn't support it
# 	GOOS=darwin GOARCH=arm64 go build -o defective
# 	butler push defective milk9111/defective:mac-arm64

web:
	wasmnow -b
	zip -r _web.zip wasmnow
	butler push _web.zip milk9111/defective:html5

web-serve:
	wasmnow 