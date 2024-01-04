FROM ubuntu:20.04

# Avoid prompts from apt-get
ENV DEBIAN_FRONTEND=noninteractive

# Install PostgreSQL and other dependencies
RUN apt-get update && apt-get install -y postgresql postgresql-contrib

# Set environment variables for PostgreSQL
ENV POSTGRES_DB=fern
ENV POSTGRES_USER=fern
ENV POSTGRES_PASSWORD=fern

# Initialize the database and create the user and database (adjust commands as needed)
RUN service postgresql start && \
    su postgres -c "createuser --superuser $POSTGRES_USER" && \
    su postgres -c "createdb --owner=$POSTGRES_USER $POSTGRES_DB" && \
    su postgres -c "psql -c \"ALTER USER $POSTGRES_USER WITH PASSWORD '$POSTGRES_PASSWORD';\""

# Set the working directory for the application
WORKDIR /usr/src/app

# Copy the compiled application binary into the container
COPY ./bin/fern_linux_amd64 /usr/src/app/fern

# Expose the port that your application uses
EXPOSE 8080
RUN mkdir config
RUN ls -l
COPY ./config/config.yaml config/
COPY wait-for-postgres.sh ./

# Define the command to run your application
# This should start PostgreSQL and then your application
CMD service postgresql start && ./wait-for-postgres.sh localhost && ./fern

