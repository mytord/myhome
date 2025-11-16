My Home
=====

Simulation of telegram webhook:
```
curl -X POST http://localhost:8080/tg/webhook          -H "Content-Type: application/json"          -d '{
   "update_id": 123456789,
   "message": {
     "message_id": 1,
     "from": {
       "id": 123456789,
       "is_bot": false,
       "first_name": "Test",
       "username": "testuser",
       "language_code": "en"
     },
     "chat": {
       "id": 123456789,
       "first_name": "Test",
       "username": "testuser",
       "type": "private"
     },
     "date": 1672531199,
     "text": "/pump_on 1"
   }
 }'
```

Renew certs:
```
sudo certbot --nginx -d yourdomain.com -d www.yourdomain.com
sudo certbot renew --dry-run
sudo certbot renew
```

Send message:
```
mosquitto_pub -h localhost -t messages -u admin -P <pwd>  -m 'Hello, world!'
```

Receive command:
```
mosquitto_sub -t commands -u admin -P <pwd> -v
```

Deploy
```
$ git pull && docker compose build && docker compose up -d www
```

Transfer
---

```
$ sudo ufw allow 80/tcp
$ sudo ufw allow 443/tcp
$ sudo apt update
$ sudo apt install certbot python3-certbot-nginx -y
$ certbot --version
```

```
$ sudo nano /etc/nginx/sites-available/<DOMAIN>

server {
    listen 80;
    server_name <DOMAIN>;

    # Для Let's Encrypt
    location /.well-known/acme-challenge/ {
        root /var/www/letsencrypt;
    }

    location / {
        proxy_pass http://localhost:<PORT>/;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
    }
}

$ sudo ln -s /etc/nginx/sites-available/<DOMAIN> /etc/nginx/sites-enabled/
$ sudo mkdir -p /var/www/letsencrypt
$ sudo chown -R www-data:www-data /var/www/letsencrypt
$ sudo nginx -t
$ sudo systemctl reload nginx
$ ...
$ sudo certbot certonly --nginx -d <DOMAIN>
```

```
$ sudo nano /etc/nginx/sites-available/<DOMAIN>

server {
    listen 443 ssl http2;
    server_name <DOMAIN>;

    ssl_certificate     /etc/letsencrypt/live/<DOMAIN>/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/<DOMAIN>/privkey.pem;

    location / {
        proxy_pass http://localhost:<PORT>/;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
    }

    location /.well-known/acme-challenge/ {
        root /var/www/letsencrypt;
    }
}

server {
    listen 80;
    server_name <DOMAIN>;

    location /.well-known/acme-challenge/ {
        root /var/www/letsencrypt;
    }

    location / {
        return 301 https://$host$request_uri;
    }
}

$ sudo nginx -t
$ sudo systemctl reload nginx
```