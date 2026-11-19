# Test

This project was generated using [Angular CLI](https://github.com/angular/angular-cli) version 22.3.0.

## Development server

To start a local development server, run:

```bash
ng serve
```

Once the server is running, open your browser and navigate to `http://localhost:4200/`. The application will automatically reload whenever you modify any of the source files.

## Code scaffolding

Angular CLI includes powerful code scaffolding tools. To generate a new component, run:

```bash
ng generate component component-name
```

For a complete list of available schematics (such as `components`, `directives`, or `pipes`), run:

```bash
ng generate --help
```

## Building

To build the project run:

```bash
ng build
```

This will compile your project and store the build artifacts in the `dist/` directory. By default, the production build optimizes your application for performance and speed.

## Running unit tests

To execute unit tests with the [Vitest](https://vitest.dev/) test runner, use the following command:

```bash
ng test
```

## Running end-to-end tests

For end-to-end (e2e) testing, run:

```bash
ng e2e
```

Angular CLI does not come with an end-to-end testing framework by default. You can choose one that suits your needs.

## Additional Resources

For more information on using the Angular CLI, including detailed command references, visit the [Angular CLI Overview and Command Reference](https://angular.dev/tools/cli) page.

## Backend VIPayment

The Go backend is located in `src/backend`. Copy `.env.example` to `.env`,
fill in the VIPayment credentials, then run:

```powershell
cd src/backend
go mod tidy
go run .
```

The API listens on port `8080` by default:

- `GET /health`
- `GET /api/services`
- `GET /api/nickname?game_code=...&user_id=...&zone_id=...`
- `POST /api/checkout`
- `GET /api/orders/:trxid/status`
- `POST /webhook/vipayment`

VIPayment requires the backend server's public IP to be whitelisted. The
webhook signature is checked using the `X-Client-Signature` header.
