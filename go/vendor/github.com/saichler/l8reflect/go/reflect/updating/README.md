# Updater

## Overview
**PUT && PATCH** operations most likely are reflecting a user action to change a model element.
With a complex & nested tree/graph model, 
it is very hard and time-consuming to maintain explicit code to apply those changes to a cache or persistence layer.  
One need to write an explicit code to attend each and every attribute.

Another challenge is reacting to explicit changes in the model and invoking specific callbacks when certain attribute change values.

## Updater
**Updater** is a utility built to extremely simplify those tasks, reducing complexity to just a few lines of code. 
Accepts an **Old/Existing** model instance and **New/Updating** model instance, 
**Updater** will apply all valid values/nested values from the **New** on the **Old**.

**Changes** that were made to the model during the update will be logged and can be used to trigger specific attribute change callbacks.

## Usage
````
//Instantiate the updater.
updater := updater.NewUpdater(common.Introspect, false)
//update the "old" instance with all the valid attributes from the "new" instance. 
err := updater.Update(old, new, common.Introspect)

//Post updating, get the list of changes.
changes := updater.Changes()
````