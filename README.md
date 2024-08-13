# Stateless Compatibility Layer

This repository contains a Dockerized Go application for the Stateless Compatibility Layer. This document provides instructions on how to set up and run the application using Docker.

## Prerequisites

Before you begin, ensure you have the following installed on your system:

- Docker
- Go (for development purposes)

## Setup

1. **Clone the repository:**

    ```sh
    git clone https://github.com/stateless-solutions/stateless-compatibility-layer.git
    cd stateless-compatibility-layer
    ```

2. **Environment Variables:**

    Create a `.env` file in the root directory of your project by copying the example file:

    ```sh
    cp .env.example .env
    ```

    Edit the `.env` file to set the necessary environment variables. The required variables are:

    ```ini
    CHAIN_URL=http://your-chain.co
    KEY_FILE=/path/to/your/.key.pem
    KEY_FILE_PASSWORD=
    IDENTITY=identity
    HTTP_PORT=8080
    CONFIG_FILES=/path/to/your/chain.json
    ```

    Ensure that the `KEY_FILE` and `CONFIG_FILES` paths point to the actual locations of your files.

## Building the Docker Image

1. **Build the Docker image:**

    Navigate to the root directory of the project and run the following command:

    ```sh
    docker build -t comp-layer .
    ```

    This command builds the Docker image and tags it as `comp-layer`.

## Running the Application

1. **Ensure your key and configuration files are in place:**

    Make sure the key file specified in the `KEY_FILE` and `CONFIG_FILES` environment variable exist and are accessible.

2. **Run the Docker container:**

    Use the following command to run the Docker container. This command reads environment variables from the `.env` file, mounts the key and configuration files, and starts the application:

    ```sh
    docker run --env-file=.env -v /path/to/your/.key.pem:/app/.key.pem -v /path/to/your/chain.json:/app/chain.json -d -p 8080:8080 --name comp-layer comp-layer --p=true
    ```

    - `--env-file=.env`: Loads the environment variables from the `.env` file.
    - `-v /path/to/your/.key.pem:/app/.key.pem`: Mounts the key file into the Docker container. Ensure this path matches the `KEY_FILE` variable in the `.env` file.
    - `-v /path/to/your/chain.json:/app/chain.json`: Mounts the config file into the Docker container. Ensure this path matches the `CONFIG_FILES` variable in the `.env` file.
    - `-d`: Runs the container in detached mode.
    - `-p 8080:8080`: Maps port 8080 on your host to port 8080 in the container. The exposed port depends on the `HTTP_PORT` environment variable.

## Accessing the Application

Once the container is up and running, the application will be accessible at `http://localhost:8080` (or the port specified in `HTTP_PORT`).

## Stopping the Application

To stop the running Docker container, use the following command:

```sh
docker stop comp-layer
```

## Removing the Container

To remove the stopped Docker container, use the following command:

```sh
docker rm comp-layer
```

## Notes

- Ensure the `CHAIN_URL`, `KEY_FILE`, `KEY_FILE_PASSWORD`, `IDENTITY`, `CONFIG_FILES` and `HTTP_PORT` environment variables are correctly set in your `.env` file.
- The `CONFIG_FILES` env var supports multiple files, just input them separated by a `,` with no spaces.
- The port specified in the `HTTP_PORT` environment variable should match the port mapping in the Docker run command.

## Troubleshooting

- **Port Conflicts**: If the specified port is already in use, you can change the `HTTP_PORT` variable in the `.env` file and update the port mapping in the Docker run command accordingly.
- **Key File and Config Files**: Ensure the file paths are correct and the files have the necessary permissions.

For further assistance, please contact the repository maintainer.

## Getting Help and Customization

This README should help you get started with running the Stateless Compatibility Layer using Docker. The Docker setup is designed to be flexible so you can deploy it however you want. Feel free to create your own Docker Compose file, Makefile, or other deployment configurations. If you encounter any issues or have questions, feel free to open an issue in the repository.
