FROM php:8.2-fpm
WORKDIR /var/www/html
RUN apt-get update && apt-get install -y git zip unzip libzip-dev libpng-dev libonig-dev libxml2-dev
RUN docker-php-ext-install pdo pdo_mysql zip mbstring xml
COPY . /var/www/html
RUN chown -R www-data:www-data /var/www/html