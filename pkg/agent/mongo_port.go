// Scaffold vacío. Define el puerto de salida ReadOnlyMongoRepository
// (CollectionInfo, FieldSample, ReadOnlyMongoRepository) siguiendo la
// sección "2. Ports de salida" de openspec/changes/add-nl-mongo-agent/tasks.md
// (tareas 2.4-2.5).
//
// INVARIANTE DE SEGURIDAD NO NEGOCIABLE: esta interfaz jamás debe declarar
// un método de escritura (Insert*, Update*, Delete*, Replace*, Drop*, etc.).
package agent
