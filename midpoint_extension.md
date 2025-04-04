Change the provider to integrate with midpoint rest API properly. Changes should affect the way changes are made via REST calls (PUT method).

Instead of sending the whole object to the API endpoint, the provider should calculate the changes from the state, and send the changes using the PATCH method, and the body should contain the changes using ObjectModificationType (form Midpoint). These are some examples:


Add attributes to the object:


```json
{
  "objectModification": {
    "itemDelta": {
      "modificationType": "add",
      "path": "description",
      "value": "Description parameter modified via REST"
    }
  }
}
```

Remove attributes from the object:

```json
{
  "objectModification": {
    "itemDelta": {
      "modificationType": "delete",
      "path": "description",
    }
  }
}
```

Replace attribute values:

```json
{
  "objectModification": {
    "itemDelta": {
      "modificationType": "replace",
      "path": "description",
      "value": "Description parameter modified via REST. Changed"
    }
  }
}
```

If multiple modifications are required, multiple calls should be made to the API endpoint each with its own itemDelta, but only if the changes are in different 1st level branches of the object.

