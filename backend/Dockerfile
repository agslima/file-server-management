FROM php:8.5-fpm

# System deps
RUN apt-get update && apt-get install -y --no-install-recommends \
    git unzip libzip-dev libpng-dev \
    && docker-php-ext-install zip pdo pdo_mysql gd \
    && rm -rf /var/lib/apt/lists/*

# Composer
COPY --from=composer:2 /usr/bin/composer /usr/bin/composer

WORKDIR /var/www

COPY . .

RUN composer install --no-interaction --prefer-dist --optimize-autoloader

RUN mkdir -p storage bootstrap/cache \
    && chown -R www-data:www-data storage bootstrap/cache

USER www-data

# Development-only entrypoint: php -S is for local/demo use and is not production-grade.
CMD ["php", "-S", "0.0.0.0:8081", "api.php"]
