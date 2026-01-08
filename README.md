# TreeSpotter Backend

The backend repository for the [Treespotter](https://github.com/Johnhi19/TreeSpotter) application. Containing all the necessary code and Dockerfiles needed to run this backend.

## Prerequisites
- Docker (desktop or engine)

For local development:
- Golang version >= 1.22.2

## Usage with docker
It is recommended to run the complete application. For that check the README.md of the main [Treespotter](https://github.com/Johnhi19/TreeSpotter) application. 

If you want only want to build and run the backend, it can be done like this:

```bash
# Build the image
docker -f Dockerfile.dev build -t backend .

# Run the container
docker run -d --name backend -p 8080:8080 backend

# Container IDs
docker ps

# Stop the container
docker stop <ID>

# Remove the container
docker rm <ID>
# While running (you have to force it):
docker rm <ID> -f

# check if it works via this url
curl http://localhost:8080/
```

The backend then runs on port 8080 of the localhost.


