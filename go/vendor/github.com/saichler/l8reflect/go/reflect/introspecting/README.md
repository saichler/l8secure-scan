# Introspection

## Overview
Introspection is an action of accepting a runtime, unknown tree/graph model, instance and deep analysis it for its structure and attributes. This is extremely helpful when building a generic purpose componet that should act on the model without having it in compile time.

An examples of components that will use the introspection, **Delta Notifications**, **ORM (Object Relation Mapping)**, **Distributed Cache**.

## Introspect
Accpets a struct instance and **Inspect** it, deep mapping the structure and attributes.

## Cloner
Deep clone a model and its instances. Will also be sensitive to model specific cloning rules, e.g. if the model has a relation of many 2 many, cloning should not clone ZSide when cloning ASide.

