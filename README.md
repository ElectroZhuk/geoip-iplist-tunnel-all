# GeoIP IP list — объединение всех групп в `all`

Этот репозиторий автоматически скачивает GeoIP-базу в формате CIDR IPv4 из OpenCCK, объединяет диапазоны IP-адресов всех исходных групп в одну категорию Xray/V2Ray с именем `all` и публикует бинарную базу вместе с контрольной суммой SHA-256 в ветку `gh-pages`.

## Источник

`https://iplist.opencck.org/?format=geoip&data=cidr4`

Параметр `cidr4` означает, что источник предоставляет **только IPv4-диапазоны**.

## Опубликованные файлы

После первого успешного запуска workflow файлы будут доступны по постоянным ссылкам:

- GeoIP-база:  
  `https://raw.githubusercontent.com/ElectroZhuk/geoip-iplist-tunnel-all/gh-pages/geoip_iplist.dat`
- Контрольная сумма SHA-256:  
  `https://raw.githubusercontent.com/ElectroZhuk/geoip-iplist-tunnel-all/gh-pages/geoip_iplist.dat.sha256`

## Автоматическая сборка

GitHub Actions запускает сборку:

- автоматически каждые 6 часов (UTC);
- при изменениях в ветке `main`;
- вручную через **Actions → Build and publish GeoIP → Run workflow**.

Workflow проверяет структуру исходного protobuf-файла и убеждается, что результат содержит ровно одну GeoIP-запись с категорией `all`.

Если скачивание, обработка или проверка завершаются ошибкой, новая версия не публикуется. Предыдущие опубликованные файлы остаются доступными.

## Использование в 3x-ui

Добавьте файл как пользовательский GeoIP-источник:

- **Тип:** GeoIP
- **Alias:** `iplist`
- **URL:** `https://raw.githubusercontent.com/ElectroZhuk/geoip-iplist-tunnel-all/gh-pages/geoip_iplist.dat`

При таком alias панель 3x-ui использует имя файла `geoip_iplist.dat`. После добавления источника обновите GeoIP-файл в панели.

Ожидаемый путь на сервере:

`/usr/local/x-ui/bin/geoip_iplist.dat`

## Использование в правилах Xray

Для обращения к пользовательской категории используйте внешний GeoIP-синтаксис:

```json
"ip": ["ext:geoip_iplist.dat:all"]
```

Убедитесь, что файл находится в каталоге geodata, который использует Xray, и что в правиле указан тот же файл.
