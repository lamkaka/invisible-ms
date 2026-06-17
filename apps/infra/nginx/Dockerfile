FROM nginx:1.27-alpine

RUN rm /etc/nginx/conf.d/default.conf

COPY apps/infra/nginx/nginx.conf /etc/nginx/nginx.conf
COPY apps/web/static/ /usr/share/nginx/html/static/
