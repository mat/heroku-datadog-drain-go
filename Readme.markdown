# heroku-datadog-drain-go

[![Build Status](https://travis-ci.org/mat/heroku-datadog-drain-go.svg)](https://travis-ci.org/mat/heroku-datadog-drain-go)

Heroku log drain forwarding metrics to Datadog. This is a liberal port of <https://github.com/ozinc/heroku-datadog-drain> to Go.

Supported Heroku metrics:

* Router response times, status codes
* Dyno runtime metrics
* Todo: Heroku Postgres metrics


## How setup a logdrain dyno


```bash
git clone git@github.com:mat/heroku-datadog-drain-go.git
cd heroku-datadog-drain-go
heroku create
heroku config:set ALLOWED_APPS=<your-app-slug> <YOUR-APP-SLUG>_PASSWORD=<password>
git push heroku master
heroku ps:scale web=1
heroku drains:add https://<your-app-slug>:<password>@<this-log-drain-app-slug>.herokuapp.com/ --app <your-app-slug>
```


## Configuration

    ALLOWED_APPS=my-app,..    # Required. Comma separated list of app names allowed to send to this drain
    <APP-NAME>_PASSWORD=..    # Required. One per allowed app where <APP-NAME> corresponds to an app name from ALLOWED_APPS


## Thanks

I wrote this together with <https://github.com/phoet> during our student exchange between <https://www.xing.com> and <http://www.jimdo.com>. Thanks for letting me work on interesting things.

## License

The MIT License (MIT)

Copyright (c) 2015 Matthias Lüdtke, Hamburg - http://github.com/mat

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
