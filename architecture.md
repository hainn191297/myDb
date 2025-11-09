# Database Architecture Based on *Fundamentals of Database Systems* and a Golang RDBMS

## I. Architectural Components and Standards

### 1. Three-Schema Architecture

This foundational architecture separates the application layer from physical storage to achieve **data independence**.

| Layer | Attributes / Standards | Description |
| --- | --- | --- |
| **External Level** | External Schemas / User Views | Describes the portion of the database a specific user group cares about and hides everything else. |
| **Conceptual Level** | Conceptual Schema | Captures the structure of the entire database for the user community, including entities, relationships, and constraints. |
| **Internal Level** | Internal Schema | Details the physical storage structure (indexes, file organization, access paths). |

### 2. Data Independence

- **Logical Data Independence:** Change the conceptual schema without modifying external schemas or application code.
- **Physical Data Independence:** Change the storage structures (internal schema) without affecting the conceptual schema.

---

## II. Data and Modeling Elements

### 1. Data and Metadata

- **Data:** Facts that can be recorded and hold contextual meaning.
- **Database:** A collection of related data describing a real-world domain (mini-world).
- **Metadata / Catalog:** Describes database structure and constraints and is used by the DBMS to understand file layouts.

### 2. Conceptual Modeling (ER Model)

| Component | Attributes | Characteristics |
| --- | --- | --- |
| **Entity** | Entity Type | Represents a real-world object (person, company, product). |
| **Attribute** | Simple/Composite, Single/Multivalued, Stored/Derived | Describes properties of an entity. |
| **Relationship** | Cardinality, Participation | Represents associations between entities (1:1, 1:N, M:N). |

### 3. Relational Model

| Component | Attributes | Characteristics |
| --- | --- | --- |
| **Relation / Table** | Set of tuples | Each tuple represents an entity or relationship instance. |
| **Attribute** | Domain | Holds atomic values drawn from a valid domain (int, string, ...). |
| **Keys** | Superkey, Candidate Key, Primary Key, Foreign Key | Guarantee uniqueness and enforce links between tables. |
| **Constraints** | Domain, Key, Entity, Referential | Define conditions a valid database must satisfy. |

---

## III. Implementation Standards

### 1. Normalization

| Normal Form | Objective | Characteristics |
| --- | --- | --- |
| **1NF** | Enforce atomic values | No multi-valued or nested attributes. |
| **2NF** | Eliminate partial functional dependencies | All non-key attributes depend on the entire primary key. |
| **3NF** | Remove transitive dependencies | Avoid indirect dependencies between non-key attributes. |
| **BCNF** | Strongest practical normalization | Remove every non-trivial functional dependency. |

### 2. System Languages and Modules

- **DDL (Data Definition Language):** Defines schemas (CREATE TABLE, ALTER, ...).
- **DML (Data Manipulation Language):** Queries and data modifications (INSERT, UPDATE, DELETE, SELECT).
- **Transactions:** Provide **ACID** guarantees:
  - *Atomicity*: All-or-nothing execution.
  - *Consistency*: Preserve valid database states.
  - *Isolation*: Transactions do not interfere.
  - *Durability*: Committed data survives failures.
- **Query Processor:** Plans and optimizes SQL requests.
- **Concurrency Control:** Manages safe concurrent access.

---

## IV. Mapping to a Golang RDBMS Architecture

A Go-based RDBMS should mirror these fundamentals within each execution layer:

| Layer in the Go RDBMS | Matching DBMS Concept | Role |
| --- | --- | --- |
| **Storage Engine** | Internal Schema | Organizes physical storage (B+Tree, WAL, file blocks). |
| **Transaction Layer** | Concurrency Control, ACID | Manages transactions and synchronizes data. |
| **Query Processor** | Query Optimization | Translates SQL into KV operations and optimizes plans. |
| **Schema Manager** | Conceptual Schema | Manages table structures, constraints, and keys. |
| **API Layer (gRPC)** | External Schema | Exposes the interface for clients and end users. |

### Architectural Metaphor

1. **Foundation (Internal/Physical):** Decides which materials exist (indexes, file structures).
2. **Frame (Conceptual):** Logical schema that defines entities and relationships.
3. **Facade (External):** What users see and interact with.
4. **DBMS Core:** The orchestrator that enforces security, recovery, query processing, and concurrency.

---

## V. Practical Application in Golang

| Go Component | Function | Equivalent in *Fundamentals* |
| --- | --- | --- |
| `btree` / `lsm` packages | File organization | Internal Schema |
| `goroutines` / `channels` | Parallel processing | Concurrency Control |
| `sync/atomic`, `sync.Mutex` | Synchronization | Transaction Isolation |
| `etcd`, `protobuf`, `grpc` | Distributed config and communication | External Schema |
| `parser` / `optimizer` modules | SQL-to-execution-plan translation | Query Processor |

---

## Conclusion

From the theoretical model in *Fundamentals of Database Systems* to a practical Golang implementation, a modern RDBMS must honor three core principles:

- **Layered abstractions.**
- **Data independence.**
- **Strict ACID guarantees with thoughtful query optimization.**

Building an RDBMS in Go becomes the exercise of realizing the three-schema model inside a highly concurrent runtime where goroutines execute transactions in parallel and channels provide safe pathways between system layers.
