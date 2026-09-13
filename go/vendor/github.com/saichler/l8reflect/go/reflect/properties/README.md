# Property

## Overview
Setting & getting nested attributes values from a complex & nested tree/graph models via metadata driven & model agnostic component, is not an easy task. 
However, the value of such component is extremely appreciated when sending **Delta Notification** and applying the **Delta Notification** on the other side is encasulated into 3-4 lines of code.

Needless to say that engineers find this challenging so they tent to dismiss this with "performance" arguments and overlook the long term maintenance & time 2 market benefits.

## The Property
**Property** is a metadata driven nested tree/graph model setter & getter utility. 
It is encapsulating the [introspect](https://github.com/saichler/my.simple/tree/main/go/introspect) modeling and use it to traverse the model instance to set & get values in a meta driven way.

For example, say you have the following model instance:
````
	myInstsnce:=model.MyTestModel{
		MyString: "Hello",
		MySingle: &model.MyTestSubModelSingle{MyString: "World"},
	}
````
And you wish, via metadata driven method to extract the value of "MyString" from within the MySingle instance, within myInstance.
````
property,_:=property.PropertyOf("mytestmodel.mysingle.mystring",introspect.DefaultIntrospect)
````
This will create an **Property** pointing to this attribute in a metadata fashion.
Now we can set/get this attribute value from/to any other instance of MyTestModel like so:
````
	//Getting a value
	v,_:=property.Get(myInstsnce)
	
	//Creating another instance
	myOtherInstance:=model.MyTestModel{}
	
	//Setting the value we fetched from the original instance
	property.Set(myOtherInstance,"Metadata")
````

This utility is extremely powerful when updating **Delta** pieces of data in a distributed cache environments with minimal 2 no effort.