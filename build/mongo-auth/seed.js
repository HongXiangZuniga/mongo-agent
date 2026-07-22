// Seed del usuario de prueba para la base de usuarios del login (authdb).
// Se ejecuta automáticamente por el contenedor mongo-auth vía
// /docker-entrypoint-initdb.d en la PRIMERA inicialización del volumen.
//
// Idempotente (upsert). El hash almacenado es bcrypt de la contraseña de
// prueba; la contraseña en texto plano NUNCA aparece aquí (se documenta solo
// en README.md / design.md).
db.users.updateOne(
  { username: "admin" },
  { $set: { username: "admin", password_hash: "$2a$10$SDCTxsKtCS.EMcrXjgtFzOofl54IPeoWHgolWDuIfz7BE/eKF4TRG" } },
  { upsert: true }
);
