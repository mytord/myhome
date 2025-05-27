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