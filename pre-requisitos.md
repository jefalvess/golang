## **Comparação de Itens - V2 - Go**

### **Objetivo**

Construa uma API backend que forneça detalhes de produtos para uso em um recurso de comparação de itens.  
Sua implementação deve seguir as melhores práticas de backend, fornecendo endpoints claros e eficientes para recuperar os dados necessários para comparações de produtos.

---

### **Requisitos**

#### **Backend: Desenvolvimento da API**

**Endpoint da API:**

* Construa uma API RESTful que retorne detalhes de múltiplos itens a serem comparados.
* A API deve fornecer campos como nome do produto, URL da imagem, descrição, preço, avaliação e especificações.
* Inclua tratamento de erros e comentários inline explicando sua lógica.

---

### **Stack**

* Você pode usar qualquer tecnologia ou framework backend de sua escolha.
* Simule a persistência de dados usando arquivos locais JSON/CSV ou um banco em memória (ex: SQLite, H2 Database) para representar o inventário. Não é necessário um banco real.

---

### **Requisitos funcionais**

O modelo de produto deve encapsular informações essenciais, incluindo, mas não se limitando aos seguintes atributos:

* ID
* nome
* descrição
* preço
* tamanho
* peso
* cor

Além disso, certos produtos podem exigir informações especializadas.  
Por exemplo, um smartphone deve incluir detalhes como:

* capacidade da bateria
* especificações da câmera
* memória
* capacidade de armazenamento
* marca
* versão do modelo
* sistema operacional

O usuário deve poder consultar comparações específicas entre itens e ignorar outros campos.  
Isso o ajudará a focar nos detalhes mais relevantes para sua análise.

---

### **Requisitos não-funcionais**

Será dada atenção especial a boas práticas em:

* tratamento de erros
* documentação
* testes
* e quaisquer outros aspectos não-funcionais relevantes que você escolher demonstrar

---

### **Documentação & visão estratégica**

Inclua um breve README ou diagrama (opcional) que explique:

* o design da sua API
* principais endpoints
* instruções de configuração
* e quaisquer decisões arquiteturais importantes tomadas durante o desenvolvimento