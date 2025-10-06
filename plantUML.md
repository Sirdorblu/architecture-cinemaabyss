@startuml
!include https://raw.githubusercontent.com/plantuml-stdlib/C4-PlantUML/master/C4_Container.puml
LAYOUT_TOP_DOWN()
title Cinemaabyss To-Be (Containers)

Person(user, "Пользователь", "мобильные приложения, web, Smart TV")

System_Boundary(ca, "CinemaAbyss Platform (Kubernetes)") {

  Container(ing, "Ingress / API Gateway (BFF)", "NGINX + Go", "Единая точка входа; маршрутизация и Strangler-proxy")
  Container(auth, "Identity & Access", "Go + JWT", "Аутентификация/авторизация,выдача токенов")
  ContainerDb(auth_db, "IAM DB", "PostgreSQL")
  Container(meta, "Metadata / Catalog", "Go (HTTP)", "Фильмы,жанры, актёры,оценки")
  ContainerDb(meta_db, "Catalog DB", "PostgreSQL")

  Container(subs, "Subscriptions", "Go (HTTP + events)", "Планы,статусы подписок")
  ContainerDb(subs_db, "Subscriptions DB", "PostgreSQL")
  Container(pay, "Payments", "Go (HTTP + events)", "Оркестрация платежей")
  ContainerDb(pay_db, "Payments DB", "PostgreSQL")
  Container(movies, "Movies API (MVP)", "Go (HTTP)", "Выделенный сервис для фильмов")
  ContainerDb(movies_db, "Movies DB", "PostgreSQL")
  Container(events, "Events API", "Go (HTTP → Kafka)", "Приём доменных событий и публикация в шину")
  ContainerQueue(bus, "Event Bus", "Apache Kafka", "movie-events,user-events,payment-events")
  Container(obj, "Object Storage", "S3-compatible", "Постеры, арт, статика")
  Container(monolith, "Монолит (Legacy)", "Go (REST)", "Оставшиеся домены;постепенно «обвивается»")
  Container_Ext(psp, "Платёжные системы", "External PSP", "webhooks")
  Container_Ext(reco, "Внешняя рекомендательная система", "External", "Получение событий/каталога,рекомендации")
  Container_Ext(partners, "Онлайн-кинотеатры / партнёры", "External", "Каталог/правила дистрибуции")
}

Rel(user, ing, "HTTPS")
Rel(ing, auth, "OIDC / JWT", "HTTPS")
Rel(ing, movies, "REST", "HTTPS")
Rel(ing, meta, "REST", "HTTPS")
Rel(ing, subs, "REST", "HTTPS")
Rel(ing, pay, "REST", "HTTPS")
Rel(ing, monolith, "REST", "HTTPS")
Rel(auth, auth_db, "R/W", "SQL")
Rel(meta, meta_db, "R/W", "SQL")
Rel(movies, movies_db, "R/W", "SQL")
Rel(subs, subs_db, "R/W", "SQL")
Rel(pay, pay_db, "R/W", "SQL")
Rel(events, bus, "Produce", "Kafka")
Rel(pay, bus, "Produce payment-events", "Kafka")
Rel(subs, bus, "Produce subscription-events", "Kafka")
Rel(movies, bus, "Consume/Produce movie-events", "Kafka")
Rel(reco, bus, "Consume", "Kafka")
Rel(pay, psp, "Charge/Refund", "HTTPS")
Rel(meta, partners, "Импорт/синхронизация каталога", "HTTPS/Batch")
Rel(meta, obj, "Чтение/загрузка постеров", "S3 API")

' наблюдаемость и CI/CD подразумеваются кластером K8s;не показаны на контейнерной схеме
@enduml

